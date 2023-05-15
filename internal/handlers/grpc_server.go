package handlers

import (
	"context"
	"errors"
	"github.com/MalyginaEkaterina/shortener/internal"
	pb "github.com/MalyginaEkaterina/shortener/internal/handlers/proto"
	"github.com/MalyginaEkaterina/shortener/internal/service"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"strconv"
)

const tokenHeader = "token"

// ShortenerServer is grpc server
type ShortenerServer struct {
	pb.UnimplementedShortenerServer
	store        storage.Storage
	service      service.Service
	baseURL      string
	deleteWorker service.DeleteWorker
}

// NewShortenerServer creates ShortenerServer
func NewShortenerServer(store storage.Storage, cfg internal.Config, service service.Service, deleteWorker service.DeleteWorker) *ShortenerServer {
	return &ShortenerServer{
		store:        store,
		service:      service,
		baseURL:      cfg.BaseURL,
		deleteWorker: deleteWorker,
	}
}

// Shorten receives request with URL and returns status CREATED and shortened URL.
// Returns status CONFLICT and shortened URL if the URL has already been shortened.
// If request does not contain a valid token new user will be created.
func (s *ShortenerServer) Shorten(ctx context.Context, in *pb.ShortenReq) (*pb.ShortenResp, error) {
	userID, err := s.getUserIDOrCreate(ctx)
	if err != nil {
		return nil, err
	}

	var response pb.ShortenResp
	ind, alreadyExists, err := s.service.AddURL(ctx, in.Url, userID)
	if err != nil {
		log.Println("Error while adding URl", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	if alreadyExists {
		response.Status = pb.ShortenStatus_CONFLICT
	} else {
		response.Status = pb.ShortenStatus_CREATED
	}
	response.Result = s.baseURL + "/" + strconv.Itoa(ind)
	return &response, nil
}

// GetUserUrls returns the list of shortened and original URLs for the user.
func (s *ShortenerServer) GetUserUrls(ctx context.Context, _ *pb.GetUserUrlsReq) (*pb.GetUserUrlsResp, error) {
	userID, err := s.getUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Unable to get userID")
	}

	var response pb.GetUserUrlsResp
	urls, err := s.store.GetUserUrls(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		log.Println("Error while getting URLs", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	for urlID, originalURL := range urls {
		response.Urls = append(response.Urls,
			&pb.ShortenedOriginalUrl{
				ShortUrl:    s.baseURL + "/" + strconv.Itoa(urlID),
				OriginalUrl: originalURL,
			})
	}
	return &response, nil
}

// ShortenBatch receives the list of URLs and their correlation_id and returns
// the list of shortened URLs with their correlation_id.
// If request does not contain a valid token a new user will be created.
func (s *ShortenerServer) ShortenBatch(ctx context.Context, in *pb.ShortenBatchReq) (*pb.ShortenBatchResp, error) {
	userID, err := s.getUserIDOrCreate(ctx)
	if err != nil {
		return nil, err
	}

	var urls []internal.CorrIDOriginalURL
	for _, v := range in.Urls {
		urls = append(urls, internal.CorrIDOriginalURL{CorrID: v.CorrelationId, OriginalURL: v.OriginalUrl})
	}

	corrIDUrlIDs, err := s.store.AddBatch(ctx, urls, userID)
	if err != nil {
		log.Println("Error while adding URls", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	var response pb.ShortenBatchResp
	for _, v := range corrIDUrlIDs {
		response.Urls = append(response.Urls,
			&pb.CorrIDShortenedUrl{
				CorrelationId: v.CorrID,
				ShortUrl:      s.baseURL + "/" + strconv.Itoa(v.URLID),
			})
	}
	return &response, nil
}

// DeleteBatch receives the list of shortened URL IDs, queued them for deletion.
func (s *ShortenerServer) DeleteBatch(ctx context.Context, in *pb.DeleteBatchReq) (*pb.DeleteBatchResp, error) {
	userID, err := s.getUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Unable to get userID")
	}

	idsToDelete := make([]internal.IDToDelete, len(in.UrlIds))

	for i, idStr := range in.UrlIds {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "Unable to parse urlIDs")
		}
		idsToDelete[i] = internal.IDToDelete{ID: id, UserID: userID}
	}
	s.deleteWorker.Delete(idsToDelete)
	return &pb.DeleteBatchResp{}, nil
}

func (s *ShortenerServer) getUserID(ctx context.Context) (int, error) {
	var token string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(tokenHeader)
		if len(values) > 0 {
			token = values[0]
		}
	}
	return s.service.GetUserID(token)
}

func (s *ShortenerServer) getUserIDOrCreate(ctx context.Context) (int, error) {
	var token string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(tokenHeader)
		if len(values) > 0 {
			token = values[0]
		}
	}
	userID, sign, err := s.service.GetUserIDOrCreate(ctx, token)
	if err != nil {
		return 0, status.Error(codes.Internal, "Internal server error")
	}

	header := metadata.New(map[string]string{tokenHeader: sign})
	if err := grpc.SendHeader(ctx, header); err != nil {
		return 0, status.Errorf(codes.Internal, "Unable to send token header")
	}
	return userID, nil
}
