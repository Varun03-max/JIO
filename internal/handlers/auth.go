package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"github.com/Varun03-max/JIO/pkg/scheduler"
	"github.com/Varun03-max/JIO/pkg/television"
	"github.com/Varun03-max/JIO/pkg/utils"
)

const (
	REFRESH_TOKEN_TASK_ID    = "jiotv_refresh_token"
	REFRESH_SSOTOKEN_TASK_ID = "jiotv_refresh_sso_token"
)

// LoginSendOTPHandler sends OTP for login
func LoginSendOTPHandler(c *fiber.Ctx) error {
	formBody := new(LoginSendOTPRequestBodyData)
	if err := c.BodyParser(formBody); err != nil {
		utils.Log.Println("Invalid JSON:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
	}
	if err := checkFieldExist("Mobile Number", formBody.MobileNumber != "", c); err != nil {
		return err
	}

	result, err := utils.LoginSendOTP(formBody.MobileNumber)
	if err != nil {
		utils.Log.Println("Send OTP Error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{"status": result})
}

// LoginVerifyOTPHandler verifies OTP and logs in the user
func LoginVerifyOTPHandler(c *fiber.Ctx) error {
	formBody := new(LoginVerifyOTPRequestBodyData)
	if err := c.BodyParser(formBody); err != nil {
		utils.Log.Println("Invalid JSON:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
	}
	if err := checkFieldExist("Mobile Number", formBody.MobileNumber != "", c); err != nil {
		return err
	}
	if err := checkFieldExist("OTP", formBody.OTP != "", c); err != nil {
		return err
	}

	result, err := utils.LoginVerifyOTP(formBody.MobileNumber, formBody.OTP)
	if err != nil {
		utils.Log.Println("Verify OTP Error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Internal server error"})
	}

	if result["status"] == "success" {
		// Only now we init the handlers safely
		if err := Init(); err != nil {
			utils.Log.Println("Init failed:", err)
		}
		return c.JSON(fiber.Map{"status": "success", "message": "Login successful"})
	}

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Login failed"})
}

// LoginPasswordHandler logs in using mobile + password
func LoginPasswordHandler(c *fiber.Ctx) error {
	var username, password string

	if c.Method() == fiber.MethodGet {
		username = c.Query("username")
		password = c.Query("password")
	} else {
		formBody := new(LoginRequestBodyData)
		if err := c.BodyParser(formBody); err != nil {
			utils.Log.Println("Invalid JSON:", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
		}
		username = formBody.Username
		password = formBody.Password
	}

	if err := checkFieldExist("Username", username != "", c); err != nil {
		return err
	}
	if err := checkFieldExist("Password", password != "", c); err != nil {
		return err
	}

	result, err := utils.Login(username, password)
	if err != nil {
		utils.Log.Println("Password login error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Internal server error"})
	}

	if result["status"] == "success" {
		if err := Init(); err != nil {
			utils.Log.Println("Init failed after password login:", err)
		}
		return c.JSON(fiber.Map{"status": "success", "message": "Login successful"})
	}

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Login failed"})
}

// LogoutHandler logs out and resets session
func LogoutHandler(c *fiber.Ctx) error {
	if !isLogoutDisabled {
		if err := utils.Logout(); err != nil {
			utils.Log.Println("Logout error:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Internal server error"})
		}
		if err := Init(); err != nil {
			utils.Log.Println("Re-init after logout error:", err)
		}
	}
	return c.Redirect("/", fiber.StatusFound)
}
