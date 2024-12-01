package handlers

import (
	userRepository "example-go-project/internal/repository/user"

	"github.com/gin-gonic/gin"
)

func NewUserHandler(userRepo userRepository.UserRepository) *UserHandler {
    return &UserHandler{
        userRepo: userRepo,
    }
}

func (h *UserHandler) Login(c *gin.Context) {
    // ... login logic ...
    // ใช้ h.authService.GenerateToken() เพื่อสร้าง token
}

func (h *UserHandler) Register(c *gin.Context) {
    // ... register logic ...
}

func (h *UserHandler) GetProfile(c *gin.Context) {
    // userID, _ := c.Get("userID")
    // ... get profile logic ...
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
    // userID, _ := c.Get("userID")
    // ... update profile logic ...
}