local function fixedWindow(key, value, size, limit)
	local counter = redis.call("get", key)
	if counter == false then
		counter = 0
	end
	if counter + value > limit then
		local v = redis.call("pttl", key)
		if v == -2 then
			v = 0
		end
		return { tonumber(counter), v }
	end
	if counter == 0 then
		redis.call("set", key, value, "px", size)
		return { tonumber(value), -1 }
	end
	return { redis.call("incrby", key, value), -1 }
end

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
		return { slidingWindowCounter, currWindowRemainingDuration }
	end
	if currWindowCounter == 0 then
		redis.call("set", currWindowKey, value, "px", size * 2)
	else
		redis.call("incrby", currWindowKey, value)
	end
	return { counter, -1 }
end

local z = 0
local limit, v, result
for i, key in ipairs(KEYS) do
	z = z + 4
	limit = tonumber(ARGV[z - 1])
	if ARGV[z] == "1" then
		v = fixedWindow(key, ARGV[z - 3], ARGV[z - 2], limit)
	else
		v = slidingWindow(key, ARGV[z - 3], ARGV[z - 2], limit)
	end
	if i == 1 then -- first result
		result = { v[1], v[2], limit };
	elseif v[2] == -1 then -- ok
		if result[2] == -1 and result[3] - result[1] > limit - v[1] then -- minimal remainder
			result = { v[1], v[2], limit };
		end
	elseif result[2] == -1 then -- not ok first time
		result = { v[1], v[2], limit };
	elseif result[2] < v[2] then -- maximum TTL
		result = { v[1], v[2], limit };
	end
end
return result