package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	baseURL       = "http://localhost:8080"
	adminUsername = "admin"
	adminPassword = "admin123"
	numConcurrent = 10               // Number of concurrent goroutines for submissions
	testDuration  = 30 * time.Second // Duration to run the test
)

// User struct to hold user info like username, password, and token
type User struct {
	Username string
	Password string
	Token    string
}

// randomString generates a random string of length n
func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// signupUsers creates a given number of users and returns them
func signupUsers(count int) []User {
	var users []User
	// Create a client with a cookie jar (to handle cookies)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	for i := 0; i < count; i++ {
		// Generate random username and fixed password
		username := "user_" + randomString(10)
		password := "veryGoodPassword"

		// Prepare the form data
		form := url.Values{}
		form.Add("username", username)
		form.Add("password", password)

		// Create the request
		req, err := http.NewRequest("POST", baseURL+"/auth/signup", strings.NewReader(form.Encode()))
		if err != nil {
			log.Println("Error creating request:", err)
			continue
		}

		// Set the headers
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,fa-IR;q=0.8,fa;q=0.7")
		req.Header.Set("Cache-Control", "max-age=0")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", baseURL)
		req.Header.Set("Referer", baseURL+"/auth/signup")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
		req.Header.Set("sec-ch-ua", `"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)

		// Send the request
		resp, err := client.Do(req)
		if err != nil {
			log.Println("Error sending request for", username, ":", err)
			continue
		}

		// Handle response
		if resp.StatusCode != http.StatusOK {
			log.Println("Signup failed for", username, "Status:", resp.Status)
		} else {
			log.Println("Signup succeeded for", username)
			// Capture the "token" cookie if it's set
			u, _ := url.Parse(baseURL)
			for _, cookie := range client.Jar.Cookies(u) {
				if cookie.Name == "token" {
					// Append user with token info
					users = append(users, User{
						Username: username,
						Password: password,
						Token:    cookie.Value,
					})
				}
			}
		}

		// Close the response body

		// Optional small delay to avoid hammering the server too fast
		//time.Sleep(10 * time.Millisecond)
	}

	return users
}

// signupUser creates a single user and returns it
func signupUser() (*User, error) {
	// Create a client with a cookie jar (to handle cookies)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	// Generate random username and fixed password
	username := "user_" + randomString(10)
	password := "veryGoodPassword"

	// Prepare the form data
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)

	// Create the request
	req, err := http.NewRequest("POST", baseURL+"/auth/signup", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,fa-IR;q=0.8,fa;q=0.7")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/auth/signup")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", `"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request for %s: %w", username, err)
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signup failed for %s with status: %s", username, resp.Status)
	}

	// Capture the "token" cookie if it's set
	u, _ := url.Parse(baseURL)
	for _, cookie := range client.Jar.Cookies(u) {
		if cookie.Name == "token" {
			// Return user with token info
			return &User{
				Username: username,
				Password: password,
				Token:    cookie.Value,
			}, nil
		}
	}

	return nil, errors.New("token was not found after signup")
}

// signupUsersConcurrent creates users concurrently and measures throughput
func signupUsersConcurrent(numConcurrent int, duration time.Duration) []User {
	var (
		wg            sync.WaitGroup
		successCount  int64
		failureCount  int64
		totalAttempts int64
		usersMutex    sync.Mutex
		users         []User
	)

	// Create a channel to signal workers to stop
	done := make(chan struct{})

	// Start the timer
	startTime := time.Now()

	// Launch worker goroutines
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
					atomic.AddInt64(&totalAttempts, 1)
					user, err := signupUser()
					if err != nil {
						atomic.AddInt64(&failureCount, 1)
						log.Printf("Worker %d: Failed to signup user: %v", workerID, err)
					} else {
						atomic.AddInt64(&successCount, 1)
						// Add the user to our collection
						usersMutex.Lock()
						users = append(users, *user)
						usersMutex.Unlock()
					}
				}
			}
		}(i)
	}

	// Wait for the test duration
	time.Sleep(duration)

	// Signal all workers to stop
	close(done)

	// Wait for all workers to finish
	wg.Wait()

	// Calculate throughput
	elapsed := time.Since(startTime)
	throughput := float64(successCount) / elapsed.Seconds()

	fmt.Printf("\n--- User Signup Load Test Results ---\n")
	fmt.Printf("Test duration: %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Successful signups: %d\n", successCount)
	fmt.Printf("Failed signups: %d\n", failureCount)
	fmt.Printf("Total attempts: %d\n", totalAttempts)
	fmt.Printf("Throughput: %.2f signups/second\n", throughput)
	fmt.Printf("Users created: %d\n", len(users))
	return users
}

func login(username, password string) (string, error) {

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)

	// Create the request - use login endpoint instead of signup
	req, err := http.NewRequest("POST", baseURL+"/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		log.Println("Error creating request:", err)
		return "", err
	}

	// Set the headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,fa-IR;q=0.8,fa;q=0.7")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/auth/signup")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", `"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request for", username, ":", err)
		return "", err
	}

	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d", resp.StatusCode)
	} else {
		u, _ := url.Parse(baseURL)
		for _, cookie := range client.Jar.Cookies(u) {
			if cookie.Name == "token" {
				return cookie.Value, nil
			}
		}
	}

	return "", errors.New("token was not found")
}

func createProblem(token string) (int, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	// Set cookies
	u, _ := url.Parse(baseURL)
	client.Jar.SetCookies(u, []*http.Cookie{
		{
			Name:  "token",
			Value: token,
		},
	})

	// Generate random problem data
	title := "Problem_" + randomString(8)
	description := "Description for " + title
	sampleInput := "sample input " + randomString(5)
	sampleOutput := "sample output " + randomString(5)
	testInput := "test input " + randomString(20)
	testOutput := "test output " + randomString(20)

	// Prepare form data
	form := url.Values{}
	form.Add("title", title)
	form.Add("description", description)
	form.Add("sample_input", sampleInput)
	form.Add("sample_output", sampleOutput)
	form.Add("time_limit", "1000")
	form.Add("memory_limit", "64000")
	form.Add("test_input_1", testInput)
	form.Add("test_output_1", testOutput)

	// Create the request
	req, err := http.NewRequest("POST", baseURL+"/problems", strings.NewReader(form.Encode()))
	if err != nil {
		return 0, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", baseURL+"/problems/form/new")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", `"Chromium";v="135", "Not-A.Brand";v="8"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Linux"`)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return 0, fmt.Errorf("problem creation failed with status %d", resp.StatusCode)
	}

	// Extract problem ID from the Location header
	// When a problem is created, the server typically redirects to the problem page
	// The URL will be something like /problems/123
	location := resp.Request.URL.Path
	if location == "" {
		return 0, errors.New("no location header found in response")
	}

	// Extract the problem ID from the URL
	parts := strings.Split(location, "/")
	if len(parts) < 3 {
		return 0, fmt.Errorf("invalid location header: %s", location)
	}

	// The last part of the URL should be the problem ID
	problemID := parts[len(parts)-1]

	// Convert the problem ID to an integer
	var id int
	_, err = fmt.Sscanf(problemID, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse problem ID from %s: %w", problemID, err)
	}

	log.Printf("Created problem %s with ID %d\n", title, id)
	return id, nil
}

func toggleProblemStatus(token string, problemID int) error {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	// Set cookies
	u, _ := url.Parse(baseURL)
	client.Jar.SetCookies(u, []*http.Cookie{
		{
			Name:  "token",
			Value: token,
		},
	})

	// Prepare form data for PUT method simulation
	form := url.Values{}
	form.Add("_method", "PUT")

	// Create the request
	toggleURL := fmt.Sprintf("%s/problems/%d/toggle-status", baseURL, problemID)
	req, err := http.NewRequest("POST", toggleURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers based on the curl command
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,fa;q=0.8,en-GB;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", baseURL+"/problems/my")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", `"Chromium";v="135", "Not-A.Brand";v="8"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Linux"`)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("toggle status failed with status %d", resp.StatusCode)
	}

	log.Printf("Successfully toggled status for problem %d\n", problemID)
	return nil
}

// submitAnswer submits a solution to a problem
func submitAnswer(token string, problemID int) error {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	// Set cookies
	u, _ := url.Parse(baseURL)
	client.Jar.SetCookies(u, []*http.Cookie{
		{
			Name:  "token",
			Value: token,
		},
	})

	// Define the boundary for multipart form data
	boundary := "----WebKitFormBoundary4fPHwUjCAPYUvn6s"

	// Create the multipart form data body
	var body strings.Builder

	// Add problem_id field
	body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	body.WriteString("Content-Disposition: form-data; name=\"problem_id\"\r\n\r\n")
	body.WriteString(fmt.Sprintf("%d\r\n", problemID))

	// Add code field with the Go solution
	body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	body.WriteString("Content-Disposition: form-data; name=\"code\"\r\n\r\n")
	body.WriteString(`package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
}
`)
	body.WriteString("\r\n")

	// Add empty file field
	body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	body.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"\"\r\n")
	body.WriteString("Content-Type: application/octet-stream\r\n\r\n\r\n")

	// Close the multipart form
	body.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	// Create the request
	submissionURL := fmt.Sprintf("%s/submissions", baseURL)
	req, err := http.NewRequest("POST", submissionURL, strings.NewReader(body.String()))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,fa;q=0.8,en-GB;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", fmt.Sprintf("%s/submissions/problem/%d/new", baseURL, problemID))
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", `"Chromium";v="135", "Not-A.Brand";v="8"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Linux"`)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("submission failed with status %d", resp.StatusCode)
	}

	log.Printf("Successfully submitted answer for problem %d\n", problemID)
	return nil
}

// submitAnswersConcurrent runs concurrent submissions to a problem and measures throughput
func submitAnswersConcurrent(users []User, problemID int, duration time.Duration) {
	if len(users) == 0 {
		fmt.Println("No users available for submissions")
		return
	}

	var (
		wg            sync.WaitGroup
		successCount  int64
		failureCount  int64
		totalAttempts int64
	)

	// Create a channel to signal workers to stop
	done := make(chan struct{})

	// Start the timer
	startTime := time.Now()

	// Launch worker goroutines
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker gets a random user from the pool
			userIndex := workerID % len(users)
			user := users[userIndex]

			for {
				select {
				case <-done:
					return
				default:
					atomic.AddInt64(&totalAttempts, 1)
					err := submitAnswer(user.Token, problemID)
					if err != nil {
						atomic.AddInt64(&failureCount, 1)
						log.Printf("Worker %d: Failed to submit answer: %v", workerID, err)
					} else {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}
		}(i)
	}

	// Wait for the test duration
	time.Sleep(duration)

	// Signal all workers to stop
	close(done)

	// Wait for all workers to finish
	wg.Wait()

	// Calculate throughput
	elapsed := time.Since(startTime)
	throughput := float64(successCount) / elapsed.Seconds()

	fmt.Printf("\n--- Load Test Results ---\n")
	fmt.Printf("Test duration: %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Successful submissions: %d\n", successCount)
	fmt.Printf("Failed submissions: %d\n", failureCount)
	fmt.Printf("Total attempts: %d\n", totalAttempts)
	fmt.Printf("Throughput: %.2f submissions/second\n", throughput)
}

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Println("Starting load test...")

	// Parse command line arguments
	if len(os.Args) > 1 && os.Args[1] == "signup" {
		// Run the signup load test
		fmt.Println("Running user signup load test...")
		fmt.Printf("Test configuration: %d concurrent workers, %.0f seconds duration\n",
			numConcurrent, testDuration.Seconds())

		// Print a countdown
		fmt.Println("Starting in 3...")
		time.Sleep(1 * time.Second)
		fmt.Println("2...")
		time.Sleep(1 * time.Second)
		fmt.Println("1...")
		time.Sleep(1 * time.Second)
		fmt.Println("GO!")

		// Run the concurrent signup test
		signupUsersConcurrent(numConcurrent, testDuration)

		fmt.Println("Signup load test finished!")
		return
	}

	// Default: Run the submission load test
	const numUsers = 100 // Create more users for concurrent testing

	fmt.Printf("Test configuration: %d concurrent workers, %.0f seconds duration\n",
		numConcurrent, testDuration.Seconds())

	// Step 1: Login as admin
	fmt.Println("Logging in as admin...")
	adminToken, err := login(adminUsername, adminPassword)
	if err != nil {
		log.Fatalf("Failed to login as admin: %v", err)
	}
	fmt.Println("Admin login successful")

	// Step 2: Create a single problem as admin
	fmt.Println("Creating test problem as admin...")
	problemID, err := createProblem(adminToken)
	if err != nil {
		log.Fatalf("Failed to create problem: %v", err)
	}
	fmt.Printf("Created problem with ID %d\n", problemID)

	// Step 3: Make the problem public
	fmt.Println("Making problem public...")
	err = toggleProblemStatus(adminToken, problemID)
	if err != nil {
		log.Printf("Failed to toggle problem status: %v", err)
	} else {
		fmt.Println("Problem is now public")
	}

	// Step 4: Create multiple users for testing
	fmt.Printf("Creating %d test users...\n", numUsers)
	users := signupUsersConcurrent(numUsers, testDuration)
	fmt.Printf("Created %d users\n", len(users))

	// Step 5: Run concurrent submissions and measure throughput
	fmt.Printf("Starting concurrent submissions to problem %d...\n", problemID)
	fmt.Printf("Test will run for %.0f seconds\n", testDuration.Seconds())

	// Print a countdown
	fmt.Println("Starting in 3...")
	time.Sleep(1 * time.Second)
	fmt.Println("2...")
	time.Sleep(1 * time.Second)
	fmt.Println("1...")
	time.Sleep(1 * time.Second)
	fmt.Println("GO!")

	// Run the concurrent test
	submitAnswersConcurrent(users, problemID, testDuration)

	fmt.Println("Submission load test finished!")
}
