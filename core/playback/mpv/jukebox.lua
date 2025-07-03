-- reporter.lua -- ROBUST VERSION with Cache Management
local msg = require 'mp.msg'
local utils = require 'mp.utils'
-- The cache is now a simple map of [filepath] -> database_id
local id_cache = {}

-- This dependency-free JSON encoder is working well.
function json_encode(val)
    local json_val
    local t = type(val)

    if t == 'string' then
        json_val = '"' .. string.gsub(val, '"', '\\"') .. '"'
    elseif t == 'number' or t == 'boolean' then
        json_val = tostring(val)
    elseif t == 'nil' then
        return nil -- Return nil to allow omitting keys
    elseif t == 'table' then
        local parts = {}
        local is_array = true
        local i = 1
        for k in pairs(val) do
            if k ~= i then is_array = false; break; end
            i = i + 1
        end

        if is_array then
            for j = 1, #val do
                local part_val = json_encode(val[j])
                if part_val then table.insert(parts, part_val) end
            end
            json_val = '[' .. table.concat(parts, ',') .. ']'
        else
            for k, v in pairs(val) do
                local part_val = json_encode(v)
                if part_val then
                    local key = '"' .. tostring(k) .. '":'
                    table.insert(parts, key .. json_encode(v))
                end
            end
            json_val = '{' .. table.concat(parts, ',') .. '}'
        end
    else
        json_val = '"' .. tostring(val) .. '"'
    end
    return json_val
end
-- ### ID CACHE MANAGEMENT FUNCTIONS ###

-- NEW: This is the primary way your Go app will "attach" an ID to a file.
-- It expects two arguments: the file path and the database ID.
function attach_id(filepath, database_id)
    if filepath and database_id then
        msg.info("Attaching ID '" .. database_id .. "' to file: " .. filepath)
        id_cache[filepath] = database_id
    else
        msg.warn("attach-id called with missing arguments.")
    end
end

-- This function clears the entire cache.
function clear_id_cache()
    msg.info("Clearing all " .. tostring(#id_cache) .. " items from ID cache.")
    id_cache = {}
end

-- This function syncs the cache with the current playlist, removing stale entries.
function sync_id_cache(name, new_playlist)
    if not new_playlist then return end
    
    local playlist_files = {}
    for _, track in ipairs(new_playlist) do
        if track.filename then
            playlist_files[track.filename] = true
        end
    end

    for path_key, _ in pairs(id_cache) do
        if not playlist_files[path_key] then
            msg.info("Removing stale ID from cache for: " .. path_key)
            id_cache[path_key] = nil
        end
    end
end


-- ### DATA PROVIDER FUNCTIONS ###

-- NEW: This function returns an ordered list of database IDs for the current playlist.
function get_playlist_ids()
    local mpv_playlist = mp.get_property_native("playlist")
    local id_list = {}
    if not mpv_playlist then return id_list end

    for _, track in ipairs(mpv_playlist) do
        if track.filename and id_cache[track.filename] then
            -- If we have an ID for this file, add it to the list.
            table.insert(id_list, id_cache[track.filename])
        else
            -- If we don't have an ID, insert null to maintain playlist order.
            table.insert(id_list, mp.null)
        end
    end
    return id_list
end

-- NEW: This function provides the ordered ID list to your Go application.
function update_playlist_ids_property()
    local id_list = get_playlist_ids()
    local json_string = json_encode(id_list)
    
    if json_string and json_string ~= "null" then
        -- Set the data on a property with a new, more descriptive name.
        mp.set_property("user-data/ext-playlist-ids", json_string)
    end
end


-- ### REGISTER EVENTS, OBSERVERS, AND SCRIPT MESSAGES ###
msg.info("Registering ID-based reporter events and observers...")
-- For your Go app to attach data and get the playlist
mp.register_script_message("attach-id", attach_id)
mp.register_script_message("update_playlist_ids_property", update_playlist_ids_property)

-- Syncs the ID cache whenever the playlist is changed (add, remove, clear)
mp.observe_property('playlist', 'native', sync_id_cache)

-- Clears the cache completely on shutdown
mp.register_event("shutdown", clear_id_cache)

update_playlist_ids_property()

msg.info("ID-based event jukebox reporter script loaded.")