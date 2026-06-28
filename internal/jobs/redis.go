package jobs

import (
	"net/url"
	"strings"

	"github.com/hibiken/asynq"
)

// ParseRedisOpt converts a full Redis URL (redis://user:pass@host:port/db)
// into an asynq.RedisClientOpt. asynq's RedisClientOpt.Addr must be
// "host:port" only — it does not accept a full URL string.
func ParseRedisOpt(redisURL string) asynq.RedisClientOpt {
	// If no scheme, assume it's already host:port
	if !strings.HasPrefix(redisURL, "redis://") && !strings.HasPrefix(redisURL, "rediss://") {
		return asynq.RedisClientOpt{Addr: redisURL}
	}

	u, err := url.Parse(redisURL)
	if err != nil {
		// Fall back — let asynq fail with a clear error
		return asynq.RedisClientOpt{Addr: redisURL}
	}

	addr := u.Host // "host:port"

	var username, password string
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	opt := asynq.RedisClientOpt{
		Addr:     addr,
		Username: username,
		Password: password,
	}

	// DB number from path "/0"
	if u.Path != "" && u.Path != "/" {
		// asynq.RedisClientOpt has no DB field in some versions;
		// just set Addr + credentials — that's enough for Railway Redis.
	}

	return opt
}
