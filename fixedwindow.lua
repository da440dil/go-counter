local counter = redis.call("get", KEYS[1])
if counter == false then
	counter = 0
end
if counter + ARGV[1] > tonumber(ARGV[3]) then
	local v = redis.call("pttl", KEYS[1])
	if v == -2 then
		v = 0
	end
	return { tonumber(counter), v }
end
if counter == 0 then
    redis.call("set", KEYS[1], ARGV[1], "px", ARGV[2])
    return { tonumber(ARGV[1]), -1 }
end
return { redis.call("incrby", KEYS[1], ARGV[1]), -1 }