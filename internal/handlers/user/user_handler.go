package handlers

import (
	"context"
	"example-go-project/internal/model"
	userRepository "example-go-project/internal/repository/user"
	"example-go-project/pkg/utils"
	"net/http"
	"os"
	"time"

	"example-go-project/internal/dto"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
    userRepo userRepository.UserRepository
}

func NewUserHandler(userRepo userRepository.UserRepository) *UserHandler {
    return &UserHandler{
        userRepo: userRepo,
    }
}

// @Summary Login endpoint
// @Description Post the API's login
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "User login"
// @Router /auth/login [post]
func (u *UserHandler) Login(c *gin.Context) {
    var req dto.LoginRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        errors := utils.FormatValidationError(err)
        if len(errors) > 0 {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": errors,
            })
            return
        }
        
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    user, err := u.userRepo.FindByEmail(ctx, req.Email)
    if err != nil {
        utils.SendError(c, http.StatusNotFound, "Failed to find user")
        return
    }

    if user == nil {
        utils.SendError(c, http.StatusUnauthorized, "Invalid email or password")
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        utils.SendError(c, http.StatusUnauthorized, "Invalid email or password")
        return
    }

    auth := utils.NewAuthHandler(os.Getenv("JWT_SECRET"), os.Getenv("JWT_EXPIRY"))
    token, err := auth.GenerateToken(user.ID.Hex(), user.Roles)
    if err != nil {
        utils.SendError(c, http.StatusInternalServerError, "Failed to generate token")
        return
    }
    res := gin.H{
        "token": token,
    }
    utils.SendSuccess(c, http.StatusOK, res, "Login successful")
}

// @Summary Register endpoint
// @Description Post the API's register
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "User registration details"
// @Router /auth/register [post]
func (u *UserHandler) Register(c *gin.Context) {
    var req dto.RegisterRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        errors := utils.FormatValidationError(err)
        if len(errors) > 0 {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": errors,
            })
            return
        }
        
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Check if email already exists
    existingUser, err := u.userRepo.FindByEmail(ctx, req.Email)
    if err != nil && err != mongo.ErrNoDocuments {
        utils.SendError(c, http.StatusInternalServerError, "Failed to check existing user")
        return
    }
    if existingUser != nil {
        utils.SendError(c, http.StatusBadRequest, "Email already exists")
        return
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        utils.SendError(c, http.StatusInternalServerError, "Failed to hash password")
        return
    }

    if len(req.Roles) == 0 {
        req.Roles = []string{string(utils.UserRole)}
    }

    // Create new user
    now := time.Now()
    user := &model.User{
        ID:        primitive.NewObjectID(),
        Name:      req.Name,
        Email:     req.Email,
        Password:  string(hashedPassword),
        Roles:     req.Roles,
        CreatedAt: now,
        UpdatedAt: now,
    }

    // Save to database
    if err := u.userRepo.Create(ctx, user); err != nil {
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    res := gin.H{
        "id":    user.ID.Hex(),
        "name":  user.Name,
        "email": user.Email,
        "roles": user.Roles,
    }
    
    utils.SendSuccess(c, http.StatusOK, res)
}

// @Summary Profile endpoint
// @Description Get the API's get profile
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer
// @Router /user/profile [get]
func (u *UserHandler) GetProfile(c *gin.Context) {
    userID, _ := c.Get("userID")
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    userIDStr, ok := userID.(string)
    if !ok {
        utils.SendError(c, http.StatusInternalServerError, "Failed to get user ID")
        return
    }

    user, err := u.userRepo.FindByID(ctx, userIDStr)
    if err != nil {
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    res := gin.H{
        "id":    user.ID.Hex(),
        "name":  user.Name,
        "email": user.Email,
        "roles": user.Roles,
    }
    
    utils.SendSuccess(c, http.StatusOK, res)
}

// @Summary Update endpoint
// @Description Get the API's update user
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "User ID"
// @Param request body dto.UpdateProfileRequest true "User update details"
// @Router /user/profile/{id} [put]
func (u *UserHandler) UpdateProfile(c *gin.Context) {
    var req dto.UpdateProfileRequest
    id := c.Param("id")

    if err := c.ShouldBindJSON(&req); err != nil {
        errors := utils.FormatValidationError(err)
        if len(errors) > 0 {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": errors,
            })
            return
        }
        
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        utils.SendError(c, http.StatusBadRequest, "Invalid ID format")
        return
    }

    user, err := u.userRepo.FindByID(ctx, objID.Hex())
    if err != nil {
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    now := time.Now()
    payload := &model.User{
        ID:        user.ID,
        Name:      req.Name,
        Email:     user.Email,
        Password:  user.Password,
        Roles:     user.Roles,
        UpdatedAt: now,
        CreatedAt: user.CreatedAt,
    }

    if err := u.userRepo.Update(ctx, payload); err != nil {
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    res := gin.H{
        "id":    payload.ID.Hex(),
        "name":  payload.Name,
        "email": payload.Email,
        "roles": payload.Roles,
    }
    
    utils.SendSuccess(c, http.StatusOK, res, "Profile updated successfully")
}

// @Summary Delete endpoint
// @Description Get the API's delete user
// @Tags admin
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "User ID"
// @Router /user/profile/{id} [delete]
func (u *UserHandler) DeleteUser(c *gin.Context) {
    id := c.Param("id")
    userID, _ := c.Get("userID")
    
    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        utils.SendError(c, http.StatusBadRequest, "Invalid ID format")
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    user, err := u.userRepo.FindByID(ctx, objID.Hex())
    if err != nil {
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    if user.ID.Hex() == userID {
        utils.SendError(c, http.StatusUnauthorized, "You cannot delete yourself")
        return
    }
    
    if err := u.userRepo.Delete(ctx, user.ID.Hex()); err != nil {
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }
    
    utils.SendSuccess(c, http.StatusOK, nil, "User deleted successfully")
}

// @Summary User list endpoint
// @Description Get the API's user list
// @Tags admin
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number (default: 1)" default(1)
// @Param pageSize query int false "Page size (default: 10)" default(10)
// @Param name query string false "Filter by user name"
// @Router /user/list [get]
func (u *UserHandler) UserList(c *gin.Context) {
    page, pageSize := utils.PaginationParams(c)

    var filter dto.UserFilter
    if err := c.ShouldBindQuery(&filter); err != nil {
        utils.SendError(c, http.StatusBadRequest, "Invalid filter parameters")
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    mongoFilter := bson.D{}
    if filter.Name != "" {
        mongoFilter = append(mongoFilter, bson.E{
            Key: "name", 
            Value: bson.D{{
                Key: "$regex", 
                Value: primitive.Regex{Pattern: filter.Name, Options: "i"},
            }},
        })
    }

    total, err := u.userRepo.Count(ctx, mongoFilter)
    if err != nil {
        utils.SendError(c, http.StatusInternalServerError, "Failed to count users: "+err.Error())
        return
    }

    opts := options.Find().
        SetSkip(int64((page - 1) * pageSize)).
        SetLimit(int64(pageSize)).
        SetSort(bson.D{{Key: "created_at", Value: -1}})

    users, err := u.userRepo.FindAll(ctx, mongoFilter, opts)
    if err != nil {
        utils.SendError(c, http.StatusInternalServerError, err.Error())
        return
    }

    response := utils.CreatePagination(page, pageSize, total, users)
    utils.SendSuccess(c, http.StatusOK, response)
}