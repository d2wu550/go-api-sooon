package member

import (
	"database/sql"
	"errors"
	"fmt"
	"go-api-sooon/app"
	"go-api-sooon/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const detailCodePrefix = "MEM02"

// Do 用switch case 對應用戶功能
func Do(c *gin.Context) {
	fmt.Println(c.Param("action"))
	switch c.Param("action") {
	case "test":
		v, _ := c.Get("memberID")
		c.JSON(http.StatusOK, gin.H{
			"s": 1,
			"c": v,
		})
		return
	case "loginHistory":
		// 使用者登入紀錄
		memberID := c.Param("mid")

		// 自己才能看自己紀錄
		chkMemberID, _ := strconv.ParseUint(memberID, 10, 64)
		if memberIDSelf, ok := c.Get("memberID"); ok == false || memberIDSelf.(uint64) != chkMemberID {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"s":       -9, // -9系統層級 APP不顯示錯誤訊息
				"errCode": app.DumpErrorCode(detailCodePrefix),
				"errMsg":  errors.New("session memberID lost").Error(),
			})
			return
		}
		prepareStr := "SELECT * FROM `sooon_db`.`member_login_log` WHERE `member_id` = ?"
		stmt, err := models.DBM.DB.Prepare(prepareStr)
		defer stmt.Close()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"s":       -9, // -9系統層級 APP不顯示錯誤訊息
				"errCode": app.DumpErrorCode(detailCodePrefix),
				"errMsg":  err.Error(),
			})
			return
		}

		rows, err := stmt.Query(memberID)
		defer rows.Close()
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"s":       -9, // -9系統層級 APP不顯示錯誤訊息
				"errCode": app.DumpErrorCode(detailCodePrefix),
				"errMsg":  err.Error(),
			})
			return
		}
		// log to stdout
		models.DBM.SQLDebug(prepareStr, memberID)

		var log []map[string]interface{}
		for rows.Next() {
			r := make(map[string]interface{})
			var _id, _loginTS int
			var _device, _createDt, _ip string
			if err := rows.Scan(&_id, &_device, &_loginTS, &_ip, &_createDt); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"s":       -1, // -9系統層級 APP不顯示錯誤訊息
					"errCode": app.DumpErrorCode(detailCodePrefix),
					"errMsg":  err.Error(),
				})
				return
			}

			r["memberID"] = _id
			r["device"] = _device
			r["loginTs"] = _loginTS
			r["ip"] = _ip
			r["createDt"] = _createDt
			log = append(log, r)
		}

		c.JSON(http.StatusOK, gin.H{
			"s":    1,
			"data": log,
		})
		return
	}
}
