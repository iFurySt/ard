package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminToken struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	Role  string `json:"role"`
}

type adminTokensFile struct {
	Version string       `json:"version"`
	Tokens  []AdminToken `json:"tokens"`
}

type adminPermission string

const (
	adminPermissionRead    adminPermission = "read"
	adminPermissionPublish adminPermission = "publish"
	adminPermissionReview  adminPermission = "review"
	adminPermissionOperate adminPermission = "operate"
)

const (
	adminRoleReader    = "reader"
	adminRolePublisher = "publisher"
	adminRoleReviewer  = "reviewer"
	adminRoleOperator  = "operator"
	adminRoleAdmin     = "admin"
)

type adminPrincipal struct {
	Name string
	Role string
}

const adminPrincipalKey = "admin_principal"

type adminAuthorizer struct {
	staticTokens []AdminToken
	tokensFile   string
	mutex        sync.RWMutex
	fileTokens   []AdminToken
	fileModTime  time.Time
	fileSize     int64
	fileLoaded   bool
}

func LoadAdminTokensFile(path string) ([]AdminToken, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file adminTokensFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if strings.TrimSpace(file.Version) != "1" {
		return nil, fmt.Errorf("admin tokens version must be 1")
	}
	if len(file.Tokens) == 0 {
		return nil, fmt.Errorf("admin tokens file must contain at least one token")
	}
	return NormalizeAdminTokens(file.Tokens)
}

func NormalizeAdminTokens(tokens []AdminToken) ([]AdminToken, error) {
	normalized := make([]AdminToken, 0, len(tokens))
	seen := map[string]struct{}{}
	for index, token := range tokens {
		name := strings.TrimSpace(token.Name)
		if name == "" {
			name = fmt.Sprintf("token-%d", index+1)
		}
		value := strings.TrimSpace(token.Token)
		if value == "" {
			return nil, fmt.Errorf("admin token %s is empty", name)
		}
		if _, ok := seen[value]; ok {
			return nil, fmt.Errorf("admin token %s duplicates another token", name)
		}
		seen[value] = struct{}{}
		role, err := normalizeAdminRole(token.Role)
		if err != nil {
			return nil, fmt.Errorf("admin token %s: %w", name, err)
		}
		normalized = append(normalized, AdminToken{
			Name:  name,
			Token: value,
			Role:  role,
		})
	}
	return normalized, nil
}

func newAdminAuthorizer(tokens []AdminToken, tokensFile string) *adminAuthorizer {
	tokensFile = strings.TrimSpace(tokensFile)
	if len(tokens) == 0 && tokensFile == "" {
		return nil
	}
	authorizer := &adminAuthorizer{staticTokens: tokens, tokensFile: tokensFile}
	if tokensFile != "" {
		_ = authorizer.reloadTokensFile(true)
	}
	return authorizer
}

func (authorizer *adminAuthorizer) authenticate(header string) (adminPrincipal, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return adminPrincipal{}, false
	}
	got := strings.TrimPrefix(header, prefix)
	for _, token := range authorizer.tokensSnapshot() {
		if subtle.ConstantTimeCompare([]byte(got), []byte(token.Token)) == 1 {
			return adminPrincipal{Name: token.Name, Role: token.Role}, true
		}
	}
	return adminPrincipal{}, false
}

func (authorizer *adminAuthorizer) require(permission adminPermission) gin.HandlerFunc {
	return func(context *gin.Context) {
		principal, ok := authorizer.authenticate(context.GetHeader("Authorization"))
		if !ok {
			context.JSON(http.StatusUnauthorized, gin.H{
				"errorCode": "UNAUTHENTICATED",
				"message":   "admin bearer token is required",
			})
			context.Abort()
			return
		}
		if !roleAllows(principal.Role, permission) {
			context.JSON(http.StatusForbidden, gin.H{
				"errorCode": "PERMISSION_DENIED",
				"message":   "admin token does not allow this operation",
			})
			context.Abort()
			return
		}
		context.Set(adminPrincipalKey, principal)
		context.Next()
	}
}

func (authorizer *adminAuthorizer) tokensSnapshot() []AdminToken {
	if authorizer.tokensFile != "" {
		_ = authorizer.reloadTokensFile(false)
	}
	authorizer.mutex.RLock()
	defer authorizer.mutex.RUnlock()
	tokens := make([]AdminToken, 0, len(authorizer.staticTokens)+len(authorizer.fileTokens))
	tokens = append(tokens, authorizer.staticTokens...)
	tokens = append(tokens, authorizer.fileTokens...)
	return tokens
}

func (authorizer *adminAuthorizer) reloadTokensFile(force bool) error {
	info, err := os.Stat(authorizer.tokensFile)
	if err != nil {
		return err
	}
	modTime := info.ModTime()
	size := info.Size()

	authorizer.mutex.RLock()
	unchanged := authorizer.fileLoaded && authorizer.fileModTime.Equal(modTime) && authorizer.fileSize == size
	authorizer.mutex.RUnlock()
	if unchanged && !force {
		return nil
	}

	authorizer.mutex.Lock()
	defer authorizer.mutex.Unlock()
	if authorizer.fileLoaded && authorizer.fileModTime.Equal(modTime) && authorizer.fileSize == size && !force {
		return nil
	}
	tokens, err := LoadAdminTokensFile(authorizer.tokensFile)
	if err != nil {
		return err
	}
	if err := validateNoDuplicateTokens(authorizer.staticTokens, tokens); err != nil {
		return err
	}
	authorizer.fileTokens = tokens
	authorizer.fileModTime = modTime
	authorizer.fileSize = size
	authorizer.fileLoaded = true
	return nil
}

func validateNoDuplicateTokens(left []AdminToken, right []AdminToken) error {
	seen := map[string]struct{}{}
	for _, token := range left {
		seen[token.Token] = struct{}{}
	}
	for _, token := range right {
		if _, ok := seen[token.Token]; ok {
			return fmt.Errorf("admin token %s duplicates another token", token.Name)
		}
		seen[token.Token] = struct{}{}
	}
	return nil
}

func normalizeAdminRole(role string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "", adminRoleAdmin:
		return adminRoleAdmin, nil
	case adminRoleReader:
		return adminRoleReader, nil
	case adminRolePublisher:
		return adminRolePublisher, nil
	case adminRoleReviewer:
		return adminRoleReviewer, nil
	case adminRoleOperator:
		return adminRoleOperator, nil
	default:
		return "", fmt.Errorf("role must be one of: reader, publisher, reviewer, operator, admin")
	}
}

func roleAllows(role string, permission adminPermission) bool {
	switch role {
	case adminRoleAdmin:
		return true
	case adminRoleReader:
		return permission == adminPermissionRead
	case adminRolePublisher:
		return permission == adminPermissionRead || permission == adminPermissionPublish
	case adminRoleReviewer:
		return permission == adminPermissionRead || permission == adminPermissionReview
	case adminRoleOperator:
		return permission == adminPermissionRead || permission == adminPermissionOperate
	default:
		return false
	}
}
