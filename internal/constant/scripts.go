package constant

// CheckAndDecrementSeatsScript is a Lua script that checks seat availability and decrements if available
const CheckAndDecrementSeatsScript = `
local flightKey = KEYS[1]
local requiredSeats = tonumber(ARGV[1])

-- Get current available seats
local availableSeats = tonumber(redis.call('GET', flightKey))
if not availableSeats then
    return -1  -- Flight not found in Redis
end

-- Check if enough seats are available
if availableSeats < requiredSeats then
    return 0  -- Not enough seats
end

-- Decrement seats
redis.call('DECRBY', flightKey, requiredSeats)
return 1  -- Success
`
