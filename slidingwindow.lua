local function slidingWindow(key, value, size, limit)
	local t = redis.call("time")
	local now = t[1] * 1000 + math.floor(t[2]/1000)
	local currWindowTime = now - now % size
	local currWindowKey = key .. ":" .. currWindowTime
	local prevWindowKey = key .. ":" .. currWindowTime - size
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
	local counter = slidingWindowCounter + value
	if counter > limit then
		return { 0, slidingWindowCounter, currWindowRemainingDuration }
	end
	if currWindowCounter == 0 then
		redis.call("set", currWindowKey, value, "px", size * 2)
	else
		redis.call("incrby", currWindowKey, value)
	end
	return { 1, counter, currWindowRemainingDuration }
end
return slidingWindow(KEYS[1], ARGV[1], ARGV[2], tonumber(ARGV[3]))