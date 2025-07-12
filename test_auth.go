// Add a test endpoint to check team authentication
func (h *TeamMemberHandlers) TestAuth(c *fiber.Ctx) error {
    // Check cookie
    token := c.Cookies("team_session")
    
    // Check locals
    member := c.Locals("teamMember")
    isTeam := c.Locals("isTeamMember")
    
    return c.JSON(fiber.Map{
        "has_cookie": token != "",
        "cookie_value": token,
        "has_member": member != nil,
        "is_team_member": isTeam,
        "path": c.Path(),
        "method": c.Method(),
    })
}