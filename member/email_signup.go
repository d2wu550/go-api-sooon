// Error code prefix
// 用戶註冊

package member

import (
	"crypto/sha256"
	"fmt"
	"go-api-sooon/app"
	"go-api-sooon/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// signupCodePrefix 錯誤代碼追蹤
const signupCodePrefix = "MEM00"

// RegInfo POST參數
type RegInfo struct {
	RegEmail string `form:"email" binding:"required"`
	Pwd      string `form:"p" binding:"required"` /* 密碼 */
	// RegNickName string `form:"nickName" binding:"required"`
	// RegBirthTs  uint32 `form:"birthday" binding:"required"`    /* 生日 */
	// CPW         string `form:"cpw" binding:"required"`         /* 再次確認密碼 */
	// Country     int    `form:"country" binding:"required"`     /* 註冊國家 預設1台灣 */
	// City        int    `form:"city" binding:"required"`        /* 註冊城市 */
	// Gender      int    `form:"gender" binding:"required"`      /* 性別 */
	// ExpectType  int    `form:"expectType" binding:"required"`  /* 尋找的旅遊類型 */
	// RegImei     string `form:"imei" binding:"required"`        /* imei */
	// ProfileShot string `form:"profileShot" binding:"required"` /* 大頭貼 */
}

// NewMemberReg POST參數
func NewMemberReg(c *gin.Context) {
	var reginfo RegInfo
	err := c.ShouldBind(&reginfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"s":       -9,
			"errMsg":  err.Error(),
			"errCode": app.DumpErrorCode(signupCodePrefix),
		})
		return
	}

	// db salt column
	salt := app.NewMd5String(3)
	// sha256雜湊
	h := sha256.New()
	h.Write([]byte(reginfo.Pwd + salt))
	hashPWD := fmt.Sprintf("%x", h.Sum(nil))

	stmtIns, err := models.DBM.DB.Prepare("INSERT IGNORE INTO `sooon_db`.`member`(`email`, `pwd`, `salt`, `ip_field`, `ipv4v6`, `create_ts`) VALUES(?, ?, ?, ?, INET6_ATON(?), ?)")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"s":        -9,
			"err_code": app.DumpErrorCode(signupCodePrefix),
			"err_msg":  err.Error(),
		})
		return
	}

	defer stmtIns.Close()

	result, err := stmtIns.Exec(reginfo.RegEmail, hashPWD, salt, c.ClientIP(), c.ClientIP(), time.Now().Unix())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"s":        -9,
			"err_code": app.DumpErrorCode(signupCodePrefix),
			"err_msg":  err.Error(),
		})
		return
	}

	newMember, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"s":        -9,
			"err_code": app.DumpErrorCode(signupCodePrefix),
			"err_msg":  err.Error(),
		})
		return
	}

	var outputMsg string
	var s int
	if newMember == 0 {
		s = -1
		outputMsg = "註冊失敗"
	} else {
		s = 1
		outputMsg = "註冊成功"
	}
	// 語系
	localizer := app.Loadi18n(c)
	translation, _ := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: outputMsg,
	})

	c.JSON(http.StatusOK, gin.H{
		"s":        s,
		"msg":      translation,
		"memberId": newMember,
	})

}
