package service

import (
	"context"
	"github.com/go-eagle/eagle/pkg/auth"
	"github.com/go-eagle/eagle/pkg/errcode"
	"github.com/google/wire"
	"time"
	"user-service/internal/ecode"
	"user-service/internal/model"
	"user-service/internal/repository"
	"user-service/internal/tasks"

	"google.golang.org/protobuf/types/known/emptypb"
	pb "user-service/api/user/v1"
)

var (
	_ pb.UserServiceServer = (*UserServiceServer)(nil)
)

var ProviderSet = wire.NewSet(NewUserServiceServer)

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer

	repo repository.UserInfoRepo
}

func NewUserServiceServer(repo repository.UserInfoRepo) *UserServiceServer {
	return &UserServiceServer{
		repo: repo,
	}
}

func (s *UserServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterReply, error) {
	var userBase *model.UserInfoModel
	userBase, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, ecode.ErrInternalError.WithDetails(
			errcode.NewDetails(map[string]interface{}{
				"msg": err.Error(),
			})).Status(req).Err()
	}
	userBase, err = s.repo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, ecode.ErrInternalError.WithDetails(
			errcode.NewDetails(map[string]interface{}{
				"msg": err.Error(),
			})).Status(req).Err()
	}
	if userBase != nil && userBase.ID > 0 {
		return nil, ecode.ErrUserIsExist.Status(req).Err()
	}

	pwd, err := auth.HashAndSalt(req.Password)
	if err != nil {
		return nil, errcode.ErrEncrypt
	}

	user, err := newUser(req.Username, req.Email, pwd)
	if err != nil {
		return nil, ecode.ErrInternalError.WithDetails(errcode.NewDetails(
			map[string]interface{}{
				"msg": err.Error(),
			})).Status(req).Err()
	}
	id, err := s.repo.CreateUserInfo(ctx, user)
	if err != nil {
		return nil, ecode.ErrInternalError.WithDetails(errcode.NewDetails(
			map[string]interface{}{
				"msg": err.Error(),
			})).Status(req).Err()
	}
	task, err := tasks.NewEmailWelcomeTask(id)

	return &pb.RegisterReply{}, nil
}
func (s *UserServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginReply, error) {
	return &pb.LoginReply{}, nil
}
func (s *UserServiceServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserReply, error) {
	return &pb.CreateUserReply{}, nil
}
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserReply, error) {
	return &pb.GetUserReply{}, nil
}
func (s *UserServiceServer) BatchGetUsers(ctx context.Context, req *pb.BatchGetUsersRequest) (*pb.BatchGetUsersReply, error) {
	return &pb.BatchGetUsersReply{}, nil
}
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserReply, error) {
	return &pb.UpdateUserReply{}, nil
}
func (s *UserServiceServer) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordReply, error) {
	return &pb.UpdatePasswordReply{}, nil
}

func newUser(username, email, password string) (*model.UserInfoModel, error) {
	return &model.UserInfoModel{
		Username:  username,
		Email:     email,
		Password:  password,
		Status:    int32(pb.StatusType_NORMAL),
		CreatedAt: time.Now().Unix(),
	}, nil
}