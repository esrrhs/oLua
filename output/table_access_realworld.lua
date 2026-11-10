-- Real-world integration test: simulates a typical game entity module

local Entity = {}
Entity.__index = Entity

function Entity.new(config)
    local self = setmetatable({}, Entity)
    self.transform = {}
    self.transform.position = {x = 0, y = 0, z = 0}
    self.transform.rotation = {x = 0, y = 0, z = 0}
    self.transform.scale = {x = 1, y = 1, z = 1}
    self.physics = {}
    self.physics.velocity = {x = 0, y = 0, z = 0}
    self.physics.acceleration = {x = 0, y = 0, z = 0}
    self.render = {}
    self.render.visible = true
    self.render.color = {r = 255, g = 255, b = 255, a = 255}
    return self
end

function Entity:update(dt)
    -- Update physics: reads self.physics.velocity and self.physics.acceleration
    self.physics.velocity.x = self.physics.velocity.x + self.physics.acceleration.x * dt
    self.physics.velocity.y = self.physics.velocity.y + self.physics.acceleration.y * dt
    self.physics.velocity.z = self.physics.velocity.z + self.physics.acceleration.z * dt

    -- Update position from velocity
    self.transform.position.x = self.transform.position.x + self.physics.velocity.x * dt
    self.transform.position.y = self.transform.position.y + self.physics.velocity.y * dt
    self.transform.position.z = self.transform.position.z + self.physics.velocity.z * dt

    -- Check bounds
    if self.transform.position.x > 1000 then
        self.transform.position.x = 1000
        self.physics.velocity.x = 0
    end
    if self.transform.position.y > 1000 then
        self.transform.position.y = 1000
        self.physics.velocity.y = 0
    end
end

function Entity:setColor(r, g, b, a)
    self.render.color.r = r
    self.render.color.g = g
    self.render.color.b = b
    self.render.color.a = a or 255
end

function Entity:getDistanceTo(other)
    local dx = self.transform.position.x - other.transform.position.x
    local dy = self.transform.position.y - other.transform.position.y
    local dz = self.transform.position.z - other.transform.position.z
    return math.sqrt(dx * dx + dy * dy + dz * dz)
end

function Entity:serialize()
    local data = {}
    data.pos_x = self.transform.position.x
    data.pos_y = self.transform.position.y
    data.pos_z = self.transform.position.z
    data.rot_x = self.transform.rotation.x
    data.rot_y = self.transform.rotation.y
    data.rot_z = self.transform.rotation.z
    data.vel_x = self.physics.velocity.x
    data.vel_y = self.physics.velocity.y
    data.vel_z = self.physics.velocity.z
    data.visible = self.render.visible
    data.color_r = self.render.color.r
    data.color_g = self.render.color.g
    data.color_b = self.render.color.b
    return data
end

function Entity:applyForce(fx, fy, fz)
    self.physics.acceleration.x = self.physics.acceleration.x + fx
    self.physics.acceleration.y = self.physics.acceleration.y + fy
    self.physics.acceleration.z = self.physics.acceleration.z + fz
end

function Entity:resetPhysics()
    self.physics.velocity.x = 0
    self.physics.velocity.y = 0
    self.physics.velocity.z = 0
    self.physics.acceleration.x = 0
    self.physics.acceleration.y = 0
    self.physics.acceleration.z = 0
end

function Entity:processCollision(other)
    local dist = self:getDistanceTo(other)
    if dist < 10 then
        -- Collision detected, reflect velocity
        self.physics.velocity.x = -self.physics.velocity.x
        self.physics.velocity.y = -self.physics.velocity.y
        -- Notify collision handler (function call invalidates)
        self:onCollision(other)
        -- After callback, velocity might have changed
        if self.physics.velocity.x > 100 then
            self.physics.velocity.x = 100
        end
        if self.physics.velocity.y > 100 then
            self.physics.velocity.y = 100
        end
    end
end

return Entity
