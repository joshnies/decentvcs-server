package models

type HasTeamAccessResponse struct {
	Team      *Team
	Role      Role
	HasAccess bool
}

// type HasTeamAccessWithRoleResponse struct {
// 	Team      *Team
// 	HasAccess bool
// 	Role      Role
// }
