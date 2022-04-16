local function fixedWindow(key, value, size, limit)
	local counter = redis.call("get", key)
	if counter == false then
		counter = 0
	end
	if counter + value > limit then
		local ttl = redis.call("pttl", key)
		if ttl == -2 then
			ttl = 0
		end
		return { 0, tonumber(counter), ttl }
	end
	if counter == 0 then
		redis.call("set", key, value, "px", size)
		return { 1, tonumber(value), tonumber(size) }
	end
	return { 1, redis.call("incrby", key, value), redis.call("pttl", key) }
end
return fixedWindow(KEYS[1], ARGV[1], ARGV[2], tonumber(ARGV[3]))