local t = redis.call("time")
local now = t[1] * 1000 + math.floor(t[2]/1000)
local size = ARGV[2]
local currWindowTime = now - now % size
local currWindowKey = KEYS[1] .. ":" .. currWindowTime
local prevWindowKey = KEYS[1] .. ":" .. currWindowTime - size
local currWindowCounter = redis.call("get", currWindowKey)
if currWindowCounter == false then
	currWindowCounter = 0
end
local prevWindowCounter = redis.call("get", prevWindowKey)
if prevWindowCounter == false then
	prevWindowCounter = 0
end
local currWindowRemainingDuration = size - (now - currWindowTime)
local slidingWindowCounter = math.floor(prevWindowCounter * (currWindowRemainingDuration / size) + currWindowCounter)
local counter = slidingWindowCounter + ARGV[1]
if counter > tonumber(ARGV[3]) then
	return { slidingWindowCounter, currWindowRemainingDuration }
end
if currWindowCounter == 0 then
	redis.call("set", currWindowKey, ARGV[1], "px", size * 2)
else
	redis.call("incrby", currWindowKey, ARGV[1])
end
return { counter, -1 }