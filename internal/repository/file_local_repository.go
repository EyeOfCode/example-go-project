package repository

import (
	"context"
	"example-go-project/internal/model"
	"example-go-project/pkg/config"
	"example-go-project/pkg/utils"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LocalFileRepository interface {
	Uploads(ctx context.Context, files []*multipart.FileHeader, user *model.User) ([]*model.FileStorage, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
	FindAll(ctx context.Context, query bson.D, opts *options.FindOptions) ([]*model.FileStorage, error)
	FindOne(ctx context.Context, query bson.M) (*model.FileStorage, error)
}

type localFileRepository struct {
	collection *mongo.Collection
	config     *config.Config
}

func NewLocalFileRepository(db *mongo.Database, config *config.Config) LocalFileRepository {
	return &localFileRepository{
		collection: db.Collection("files"),
		config:     config,
	}
}

func (r *localFileRepository) Uploads(ctx context.Context, files []*multipart.FileHeader, user *model.User) ([]*model.FileStorage, error) {
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, err
	}

	var filesInfo []*model.FileStorage

	for _, file := range files {
		changeFile, err := utils.GenerateRandomFilename(file.Filename)
		if err != nil {
			return nil, err
		}
		dst := filepath.Join(uploadDir, changeFile)

		src, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer src.Close()

		dst_file, err := os.Create(dst)
		if err != nil {
			return nil, err
		}
		defer dst_file.Close()

		buffer := make([]byte, file.Size)
		n, err := io.ReadFull(src, buffer)
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(dst, buffer[:n], 0644); err != nil {
			return nil, err
		}
		var core string = "http"
		if r.config.ServerState == "production" {
			core = "https"
		}
		basePath := fmt.Sprintf("%s://%s/%s", core, r.config.BaseUrl, strings.Split(uploadDir, "/")[1])
		payload := &model.FileStorage{
			Original: file.Filename,
			Name:     changeFile,
			BasePath: basePath,
			Dir:      uploadDir,
			UserID:   user.ID,
		}
		resFileStore, err := r.collection.InsertOne(ctx, payload)
		if err != nil {
			return nil, err
		}
		payload.ID = resFileStore.InsertedID.(primitive.ObjectID)
		filesInfo = append(filesInfo, payload)
	}

	return filesInfo, nil
}

func (r *localFileRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	var fileStorage model.FileStorage
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&fileStorage)
	if err != nil {
		return err
	}

	filePath := filepath.Join(fileStorage.Dir, fileStorage.Name)
	if err := os.Remove(filePath); err != nil {
		return err
	}
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *localFileRepository) FindAll(ctx context.Context, query bson.D, opts *options.FindOptions) ([]*model.FileStorage, error) {
	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var files []*model.FileStorage
	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}

	return files, nil
}

func (r *localFileRepository) FindOne(ctx context.Context, query bson.M) (*model.FileStorage, error) {
	var fileStorage model.FileStorage
	err := r.collection.FindOne(ctx, query).Decode(&fileStorage)
	if err != nil {
		return nil, err
	}
	return &fileStorage, nil
}
