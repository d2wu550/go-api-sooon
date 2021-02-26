// Package member 會員
// JWT 會在此驗證發行token
// Sessions 會初始化並塞入用戶資料
package member

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"go-api-sooon/app"
	"go-api-sooon/config"

	"go-api-sooon/models"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// loginCodePrefix ...
const loginCodePrefix = "MEM01"

// LoginBody POST參數
type LoginBody struct {
	Email  string `form:"email" binding:"required"`
	RawPWD string `form:"p" binding:"required"`
	Lang   string `form:"lang"`
	Device string `form:"client" binding:"required"`
}

// Login ...
func Login(c *gin.Context) {
	// form body parameter
	var loginBody LoginBody
	err := c.ShouldBind(&loginBody)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"s":       -9, // -9系統層級 APP不顯示錯誤訊息
			"errCode": app.SFunc.DumpErrorCode(loginCodePrefix),
			"errMsg":  err.Error(),
		})
		return
	}

	// 查email
	ch := make(chan uint64) // member_id
	errch := make(chan error)
	tokenString := make(chan string) // jwt
	var email, pwd, salt string
	var memberID uint64
	go func() {
		{
			prepareStr := "SELECT `member_id`, `email`, `pwd`, `salt` FROM `sooon_db`.`member` WHERE `email` = ?"
			stmt, err := models.DBM.DB.Prepare(prepareStr)
			defer stmt.Close()
			if err != nil {
				errch <- err // 錯誤跳出
				return
			}

			err = stmt.QueryRow(loginBody.Email).Scan(&memberID, &email, &pwd, &salt)
			if err != nil {
				errch <- err // 錯誤跳出
				close(errch)
				return
			}
			// log to stdout
			models.DBM.SQLDebug(prepareStr, memberID, loginBody.Device, time.Now().Unix(), c.ClientIP())

			// 驗證密碼
			h := sha256.New()
			h.Write([]byte(loginBody.RawPWD + salt))
			if pwd == fmt.Sprintf("%x", h.Sum(nil)) {
				ch <- memberID // 傳給jwt payload
				close(ch)
			} else {
				fmt.Println(pwd)
				fmt.Printf("%x", h.Sum(nil))
				errch <- sql.ErrNoRows // 密碼錯故意使用sql.ErrNoRows顯示帳密錯誤
				close(errch)
				return
			}

		}
		// 更新使用者登入Log
		{
			prepareStr := "INSERT INTO `sooon_db`.`member_login_log`(`member_id`, `client_device`, `login_ts`, `ip`) VALUES (?, ?, ?, ?)"
			stmt, err := models.DBM.DB.Prepare(prepareStr)
			if err != nil {
				// 非致命錯誤 可以寄信通知或是寫入redis做定期排查
				fmt.Println(app.SFunc.DumpErrorCode(loginCodePrefix) + err.Error())
				return
			}
			defer stmt.Close()
			_, err = stmt.Exec(memberID, loginBody.Device, time.Now().Unix(), c.ClientIP())
			if err != nil {
				// 非致命錯誤 可以寄信通知或是寫入redis做定期排查
				fmt.Println(app.SFunc.DumpErrorCode(loginCodePrefix) + err.Error())
				return
			}
			// log to stdout
			models.DBM.SQLDebug(prepareStr, memberID, loginBody.Device, time.Now().Unix(), c.ClientIP())
		}
	}()

	go func() {
		_memberID := <-ch
		{ // 更新sessions
			session := sessions.Default(c)
			lang := c.Request.FormValue("lang")
			if len(lang) <= 0 {
				_membersessions := session.Get(_memberID)
				if _membersessions != nil {
					lang = _membersessions.(config.MemberSessions).Lang
				} else {
					lang = "zh"
				}
			}
			session.Set(_memberID, config.MemberSessions{
				LoginTs: time.Now().Unix(),
				Lang:    lang,
				Email:   email,
			})
			session.Save()
		}

		// 先做JWT 上面用戶編號拿到
		_, token, err := config.CreateJWTClaims(_memberID, loginBody.Email, "member", "login")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"s":       -9,
				"errCode": app.SFunc.DumpErrorCode(loginCodePrefix),
				"errMsg":  err.Error(),
			})
			return
		}
		tokenString <- token
	}()

	// 不設定default讓select強制等待goroutine
	select {
	case _token := <-tokenString: // DB驗證登入
		// 多語
		translation := app.SFunc.Localizer(c, "登入成功")
		c.JSON(http.StatusOK, gin.H{
			"s":      1,
			"member": memberID,
			"token":  _token,
			"msg":    translation,
		})
		return
	case err := <-errch:
		s := -1
		switch {
		case err == sql.ErrNoRows:
			s = -1
			err = errors.New(app.SFunc.Localizer(c, "帳號不存在"))
		case err != nil:
			s = -9
		}

		// DB initialization failed
		c.JSON(http.StatusOK, gin.H{
			"s":       s,
			"errCode": app.SFunc.DumpErrorCode(loginCodePrefix),
			"errMsg":  err.Error(),
		})

		return
	case <-time.After(time.Second * 3):
		c.JSON(http.StatusOK, gin.H{
			"s":       -9,
			"errCode": app.SFunc.DumpErrorCode(loginCodePrefix),
			"errMsg":  errors.New("Timeout").Error(),
		})
		return
		// default:
		// 	// DB initialization failed
		// 	c.JSON(http.StatusOK, gin.H{
		// 		"s":       -9,
		// 		"errCode": app.SF.DumpErrorCode(loginCodePrefix),
		// 		"errMsg":  "DB initialization failed",
		// 	})
	}
}
