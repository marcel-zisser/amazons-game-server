package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var secret = []byte(os.Getenv("JWT_SIGNING_KEY"))

func GenerateJWT(teamName string, duration time.Duration) (string, error) {
	// 1. Define the claims (payload data)
	claims := jwt.MapClaims{
		"team": teamName,                        // Custom claim: your game logic reads this
		"iat":  time.Now().Unix(),               // Issued At
		"exp":  time.Now().Add(duration).Unix(), // Expiration Time
		"iss":  "amazons-tournament-server",     // Issuer
	}

	// 2. Create the token object using HS256 (HMAC-SHA256) signing method
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 3. Sign the token with your server's secret key
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func validateToken(tokenString string) (string, error) {
	// Parse the token and verify the signature using our secret key
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token is using the expected signing method (HMAC)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return "", status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Extract the team name from the token claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		teamName, exists := claims["team"].(string)
		if !exists {
			return "", status.Errorf(codes.Unauthenticated, "token missing team claim")
		}
		return teamName, nil
	}

	return "", status.Errorf(codes.Unauthenticated, "invalid token claims")
}

// extractToken gets the Bearer token from the gRPC metadata headers
func extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	authHeader, ok := md["authorization"]
	if !ok || len(authHeader) == 0 {
		return "", status.Errorf(codes.Unauthenticated, "authorization token is required")
	}

	// Expecting "Bearer <token>"
	parts := strings.Split(authHeader[0], " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", status.Errorf(codes.Unauthenticated, "authorization header format must be 'Bearer <token>'")
	}

	return parts[1], nil
}

// AuthUnaryInterceptor guards single request/response calls
func AuthUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	token, err := extractToken(ctx)
	if err != nil {
		return nil, err
	}

	teamName, err := validateToken(token)
	if err != nil {
		return nil, err
	}

	// Optional: Inject the team name into the context so your game logic knows who called it
	newCtx := context.WithValue(ctx, "team", teamName)

	// Execute the actual RPC handler
	return handler(newCtx, req)
}

// WrappedStream allows us to inject our updated context into the stream
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// AuthStreamInterceptor guards persistent streaming connections
func AuthStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()
	token, err := extractToken(ctx)
	if err != nil {
		return err
	}

	teamName, err := validateToken(token)
	if err != nil {
		return err
	}

	// Inject the team name into the context
	newCtx := context.WithValue(ctx, "team", teamName)

	// Pass the wrapped stream with the new context to the handler
	return handler(srv, &wrappedStream{ServerStream: ss, ctx: newCtx})
}
