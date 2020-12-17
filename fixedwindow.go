package counter

import (
	"time"
)

var fwsrc = `
local counter = redis.call("get", KEYS[1])
if counter == false then
	counter = 0
end
if counter + ARGV[1] > tonumber(ARGV[3]) then
	return { tonumber(counter), redis.call("pttl", KEYS[1]) }
end
if counter == 0 then
    redis.call("set", KEYS[1], ARGV[1], "px", ARGV[2])
    return { tonumber(ARGV[1]), -1 }
end
return { redis.call("incrby", KEYS[1], ARGV[1]), -1 }
`

// FixedWindow creates new counter which implements distributed rate limiting using fixed window algorithm.
func FixedWindow(client RedisClient, size time.Duration, limit int) *Counter {
	return newCounter(client, size, limit, fwsrc)
}
