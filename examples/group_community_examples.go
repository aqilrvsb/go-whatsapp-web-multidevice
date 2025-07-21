package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const baseURL = "http://localhost:3000"

// Example structures
type CreateGroupRequest struct {
	Title        string   `json:"title"`
	Participants []string `json:"participants"`
}

type CreateCommunityRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AddParticipantsRequest struct {
	GroupID      string   `json:"group_id,omitempty"`
	CommunityID  string   `json:"community_id,omitempty"`
	Participants []string `json:"participants"`
}

type LinkGroupRequest struct {
	CommunityID string `json:"community_id"`
	GroupID     string `json:"group_id"`
}

func main() {
	// Example 1: Create a group with participants
	fmt.Println("=== Creating Group with Participants ===")
	createGroupExample()
	
	// Example 2: Add participants to existing group
	fmt.Println("\n=== Adding Participants to Existing Group ===")
	addParticipantsToGroupExample()
	
	// Example 3: Create a community
	fmt.Println("\n=== Creating Community ===")
	createCommunityExample()
	
	// Example 4: Add participants to community
	fmt.Println("\n=== Adding Participants to Community ===")
	addParticipantsToCommunityExample()
	
	// Example 5: Link group to community
	fmt.Println("\n=== Linking Group to Community ===")
	linkGroupToCommunityExample()
}

func createGroupExample() {
	request := CreateGroupRequest{
		Title: "Development Team",
		Participants: []string{
			"+1234567890",
			"+0987654321",
		},
	}
	
	response := makeRequest("POST", "/group", request)
	fmt.Println("Response:", response)
}

func addParticipantsToGroupExample() {
	request := AddParticipantsRequest{
		GroupID: "123456789@g.us", // Replace with actual group ID
		Participants: []string{
			"+1111111111",
			"+2222222222",
		},
	}
	
	response := makeRequest("POST", "/group/participants", request)
	fmt.Println("Response:", response)
}

func createCommunityExample() {
	request := CreateCommunityRequest{
		Name:        "Tech Community",
		Description: "A community for tech enthusiasts",
	}
	
	response := makeRequest("POST", "/community", request)
	fmt.Println("Response:", response)
}

func addParticipantsToCommunityExample() {
	request := AddParticipantsRequest{
		CommunityID: "987654321@g.us", // Replace with actual community ID
		Participants: []string{
			"+3333333333",
			"+4444444444",
		},
	}
	
	response := makeRequest("POST", "/community/participants", request)
	fmt.Println("Response:", response)
}

func linkGroupToCommunityExample() {
	request := LinkGroupRequest{
		CommunityID: "987654321@g.us", // Replace with actual community ID
		GroupID:     "123456789@g.us",  // Replace with actual group ID
	}
	
	response := makeRequest("POST", "/community/link-group", request)
	fmt.Println("Response:", response)
}

// Helper function to make HTTP requests
func makeRequest(method, endpoint string, payload interface{}) string {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("Error marshaling request: %v", err)
	}
	
	req, err := http.NewRequest(method, baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Sprintf("Error creating request: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error making request: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err)
	}
	
	return string(body)
}
