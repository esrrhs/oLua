-- Real-world integration test: simulates a typical game entity module

local Entity = {}
Entity.__index = Entity

function Entity.new(config)
    local self = setmetatable({}, Entity)
    self.transform = {}
    local self_transform = self.transform -- opt by oLua
    self_transform.position = {x = 0, y = 0, z = 0}
    self_transform.rotation = {x = 0, y = 0, z = 0}
    self_transform.scale = {x = 1, y = 1, z = 1}
    self.physics = {}
    local self_physics = self.physics -- opt by oLua
    self_physics.velocity = {x = 0, y = 0, z = 0}
    self_physics.acceleration = {x = 0, y = 0, z = 0}
    self.render = {}
    local self_render = self.render -- opt by oLua
    self_render.visible = true
    self_render.color = {r = 255, g = 255, b = 255, a = 255}
    return self
end

function Entity:update(dt)
    -- Update physics: reads self.physics.velocity and self.physics.acceleration
    local self_physics_velocity = self.physics.velocity -- opt by oLua
    local self_physics_acceleration = self.physics.acceleration -- opt by oLua
    self_physics_velocity.x = self_physics_velocity.x + self_physics_acceleration.x * dt
    self_physics_velocity.y = self_physics_velocity.y + self_physics_acceleration.y * dt
    self_physics_velocity.z = self_physics_velocity.z + self_physics_acceleration.z * dt

    -- Update position from velocity
    local self_transform_position = self.transform.position -- opt by oLua
    self_transform_position.x = self_transform_position.x + self_physics_velocity.x * dt
    self_transform_position.y = self_transform_position.y + self_physics_velocity.y * dt
    self_transform_position.z = self_transform_position.z + self_physics_velocity.z * dt

    -- Check bounds
    if self_transform_position.x > 1000 then
        self_transform_position.x = 1000
        self_physics_velocity.x = 0
    end
    if self_transform_position.y > 1000 then
        self_transform_position.y = 1000
        self_physics_velocity.y = 0
    end
end

function Entity:setColor(r, g, b, a)
    local self_render_color = self.render.color -- opt by oLua
    self_render_color.r = r
    self_render_color.g = g
    self_render_color.b = b
    self_render_color.a = a or 255
end

function Entity:getDistanceTo(other)
    local other_transform_position = other.transform.position -- opt by oLua
    local self_transform_position = self.transform.position -- opt by oLua
    local dx = self_transform_position.x - other_transform_position.x
    local dy = self_transform_position.y - other_transform_position.y
    local dz = self_transform_position.z - other_transform_position.z
    return math.sqrt(dx * dx + dy * dy + dz * dz)
end

function Entity:serialize()
    local data = {}
    local self_transform_position = self.transform.position -- opt by oLua
    data.pos_x = self_transform_position.x
    data.pos_y = self_transform_position.y
    data.pos_z = self_transform_position.z
    local self_transform_rotation = self.transform.rotation -- opt by oLua
    data.rot_x = self_transform_rotation.x
    data.rot_y = self_transform_rotation.y
    data.rot_z = self_transform_rotation.z
    local self_physics_velocity = self.physics.velocity -- opt by oLua
    data.vel_x = self_physics_velocity.x
    data.vel_y = self_physics_velocity.y
    data.vel_z = self_physics_velocity.z
    data.visible = self.render.visible
    local self_render_color = self.render.color -- opt by oLua
    data.color_r = self_render_color.r
    data.color_g = self_render_color.g
    data.color_b = self_render_color.b
    return data
end

function Entity:applyForce(fx, fy, fz)
    local self_physics_acceleration = self.physics.acceleration -- opt by oLua
    self_physics_acceleration.x = self_physics_acceleration.x + fx
    self_physics_acceleration.y = self_physics_acceleration.y + fy
    self_physics_acceleration.z = self_physics_acceleration.z + fz
end

function Entity:resetPhysics()
    local self_physics_velocity = self.physics.velocity -- opt by oLua
    self_physics_velocity.x = 0
    self_physics_velocity.y = 0
    self_physics_velocity.z = 0
    local self_physics_acceleration = self.physics.acceleration -- opt by oLua
    self_physics_acceleration.x = 0
    self_physics_acceleration.y = 0
    self_physics_acceleration.z = 0
end

function Entity:processCollision(other)
    local dist = self:getDistanceTo(other)
    if dist < 10 then
        -- Collision detected, reflect velocity
        local self_physics_velocity = self.physics.velocity -- opt by oLua
        self_physics_velocity.x = -self_physics_velocity.x
        self_physics_velocity.y = -self_physics_velocity.y
        -- Notify collision handler (function call invalidates)
        self:onCollision(other)
        -- After callback, velocity might have changed
        self_physics_velocity = self.physics.velocity -- opt by oLua
        if self_physics_velocity.x > 100 then
            self_physics_velocity.x = 100
        end
        if self_physics_velocity.y > 100 then
            self_physics_velocity.y = 100
        end
    end
end

return Entity
