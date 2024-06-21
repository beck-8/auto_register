package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
)

func showRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func registerHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	authCode := c.PostForm("auth_code")

	if username == "" || password == "" {
		c.String(http.StatusBadRequest, "The username and password cannot be empty.")
		return
	}

	var codeType int
	var usedBy string
	err := db.QueryRow("SELECT type, used_by FROM authcodes WHERE code = ?", authCode).Scan(&codeType, &usedBy)
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "Invalid or used auth code")
		return
	}

	if usedBy != "" {
		c.String(http.StatusBadRequest, "Auth code already used")
		return
	}

	duration := time.Duration(codeType) * 24 * time.Hour

	now := time.Now()
	expiryDate := now.Add(duration)

	_, err = db.Exec("INSERT INTO users (username, password, register_date, expiry_date, update_date, auth_code, register_ip) VALUES (?, ?, ?, ?, ?, ?, ?)",
		username, password, now.Format(time.RFC3339), expiryDate.Format(time.RFC3339), now.Format(time.RFC3339), authCode, c.ClientIP())
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "User already exists")
		return
	}

	_, err = db.Exec("UPDATE authcodes SET used_by = ?, used_date = ? WHERE code = ?", username, now.Format(time.RFC3339), authCode)
	if err != nil {
		log.Println(err)
		c.String(http.StatusInternalServerError, "Failed to update auth code")
		return
	}

	c.Redirect(http.StatusSeeOther, "/query")
	restart <- 1
}

func showQueryPage(c *gin.Context) {
	c.HTML(http.StatusOK, "query.html", nil)
}

func queryHandler(c *gin.Context) {

	username := c.PostForm("username")
	if username == "" {
		c.String(http.StatusBadRequest, "Username is required")
		return
	}

	var expiryDate string
	err := db.QueryRow("SELECT expiry_date FROM users WHERE username = ?", username).Scan(&expiryDate)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	c.HTML(http.StatusOK, "result.html", gin.H{
		"Username":   username,
		"ExpiryDate": expiryDate,
	})
}

func showRenewPage(c *gin.Context) {
	c.HTML(http.StatusOK, "renew.html", nil)
}

func renewHandler(c *gin.Context) {
	username := c.PostForm("username")
	authCode := c.PostForm("auth_code")

	var codeType int
	var usedBy string
	err := db.QueryRow("SELECT type, used_by FROM authcodes WHERE code = ?", authCode).Scan(&codeType, &usedBy)
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "Invalid or used auth code")
		return
	}

	if usedBy != "" {
		c.String(http.StatusBadRequest, "Auth code already used")
		return
	}

	duration := time.Duration(codeType) * 24 * time.Hour

	var currentExpiryDate string
	err = db.QueryRow("SELECT expiry_date FROM users WHERE username = ?", username).Scan(&currentExpiryDate)
	if err != nil {
		log.Println(err)
		c.String(http.StatusNotFound, "User not found")
		return
	}

	currentExpiry, _ := time.Parse(time.RFC3339, currentExpiryDate)
	if time.Now().After(currentExpiry) {
		currentExpiry = time.Now()
	}
	newExpiryDate := currentExpiry.Add(duration)

	_, err = db.Exec("UPDATE users SET expiry_date = ?, update_date = ? , auth_code = ? WHERE username = ?",
		newExpiryDate.Format(time.RFC3339), time.Now().Format(time.RFC3339), authCode, username)
	if err != nil {
		log.Println(err)
		c.String(http.StatusInternalServerError, "Failed to update user")
		return
	}

	_, err = db.Exec("UPDATE authcodes SET used_by = ?, used_date = ? WHERE code = ?", username, time.Now().Format(time.RFC3339), authCode)
	if err != nil {
		log.Println(err)
		c.String(http.StatusInternalServerError, "Failed to update auth code")
		return
	}

	c.Redirect(http.StatusSeeOther, "/query")
	restart <- 1
}

func updatePasswordFile() {
	// 每24小时重启一次，更新过期的账户
	go func() {
		time.Sleep(24 * time.Hour)
		c := exec.Command("systemctl", "restart", "mitmdump2")
		c.Run()
	}()

	for {
		// 每10分钟就更新文件
		time.Sleep(10 * time.Minute)

		rows, err := db.Query("SELECT username, password FROM users WHERE expiry_date > ?", time.Now().Format(time.RFC3339))
		if err != nil {
			log.Println("Failed to query users:", err)
			continue
		}

		file, err := os.Create(passwdPath)
		if err != nil {
			log.Println("Failed to create passwd.txt:", err)
			continue
		}

		for rows.Next() {
			var username, password string
			err = rows.Scan(&username, &password)
			if err != nil {
				log.Println("Failed to scan row:", err)
				continue
			}

			_, err = file.WriteString(username + ":" + password + "\n")
			if err != nil {
				log.Println("Failed to write to file:", err)
				continue
			}
		}

		rows.Close()
		file.Close()

		log.Println("update passwd success")

		// 有人提交时重启mitmdump 使其生效，最晚10分钟
		if l := len(restart); l > 0 {
			cmd := exec.Command("systemctl", "restart", "mitmdump2")
			_, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("Failed to restart", err)
			}
			log.Println("Success to restart")

			for i := 0; i < l; i++ {
				<-restart
			}
		}
	}
}
