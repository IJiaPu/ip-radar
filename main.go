package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Email struct {
		From     string `json:"from"`
		Password string `json:"password"`
		To       string `json:"to"`
		SmtpHost string `json:"smtpHost"`
		SmtpPort string `json:"smtpPort"`
	} `json:"email"`
}

// IPInfo represents information about a network interface and its IP
type IPInfo struct {
	InterfaceName string
	IPAddress     string
	Type          string // "IPv4" or "IPv6"
}

var (
	previousIPs = make(map[string]string)
	config      Config
)

func main() {
	// Check if config.json exists, create it with default values if not
	ensureConfigExists("config.json")
	// Load configuration
	err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Start web server for configuration in a separate goroutine
	go startWebServer()

	fmt.Println("IP Radar is running...")
	fmt.Println("Configuration interface available at http://localhost:8087")

	// Initial IP check
	checkIPChanges()

	// Periodically check for IP changes
	for {
		time.Sleep(10 * time.Minute)
		checkIPChanges()
	}
}

func loadConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	return err
}

func saveConfig(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(config)
	return err
}

func checkIPChanges() {
	currentIPs := getRealIPs()

	// Check for new or changed IPs
	var newIPs []IPInfo
	for _, ipInfo := range currentIPs {
		key := ipInfo.InterfaceName + "-" + ipInfo.IPAddress
		if _, exists := previousIPs[key]; !exists {
			fmt.Printf("New IP detected: %s (%s)\n", ipInfo.IPAddress, ipInfo.InterfaceName)
			newIPs = append(newIPs, ipInfo)
			previousIPs[key] = ipInfo.Type
		}
	}

	// If there are new IPs, send a notification with all of them
	if len(newIPs) > 0 {
		sendEmailNotification(newIPs)
	}
}

func getRealIPs() []IPInfo {
	var ips []IPInfo
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("Error getting network interfaces:", err)
		return ips
	}

	for _, iface := range interfaces {
		// Skip loopback, point-to-point, and down interfaces
		if iface.Flags&net.FlagLoopback != 0 ||
			iface.Flags&net.FlagPointToPoint != 0 ||
			iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Skip virtual interfaces (VMware, VirtualBox, etc.)
		if strings.Contains(strings.ToLower(iface.Name), "vmware") ||
			strings.Contains(strings.ToLower(iface.Name), "virtual") ||
			strings.Contains(strings.ToLower(iface.Name), "vbox") {
			continue
		}

		// Only include wired (Ethernet) and wireless (WLAN) interfaces
		isRealInterface := strings.Contains(strings.ToLower(iface.Name), "eth") ||
			strings.Contains(strings.ToLower(iface.Name), "en") ||
			strings.Contains(strings.ToLower(iface.Name), "wlan") ||
			strings.Contains(strings.ToLower(iface.Name), "wi-fi") ||
			strings.Contains(strings.ToLower(iface.Name), "wireless")

		if !isRealInterface {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Println("Error getting addresses for interface:", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip loopback, link-local, and private addresses
			if ip == nil || !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}

			// Skip IPv6 link-local addresses
			if ip.To4() == nil && strings.HasPrefix(ip.String(), "fe80:") {
				continue
			}

			ipType := "IPv4"
			if ip.To4() == nil {
				ipType = "IPv6"
			}

			ipInfo := IPInfo{
				InterfaceName: iface.Name,
				IPAddress:     ip.String(),
				Type:          ipType,
			}

			ips = append(ips, ipInfo)
		}
	}

	return ips
}

func sendEmailNotification(newIPs []IPInfo) {
	from := config.Email.From
	password := config.Email.Password
	to := config.Email.To
	smtpHost := config.Email.SmtpHost
	smtpPort := config.Email.SmtpPort

	// Create HTML email content
	var htmlContent strings.Builder
	htmlContent.WriteString(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            color: #333;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #f9f9f9;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        h1 {
            color: #2c3e50;
            border-bottom: 1px solid #eee;
            padding-bottom: 10px;
        }
        .ip-item {
            background-color: #fff;
            padding: 15px;
            margin-bottom: 10px;
            border-radius: 4px;
            border-left: 4px solid #3498db;
        }
        .ip-address {
            font-weight: bold;
            color: #3498db;
        }
        .interface-name {
            color: #7f8c8d;
            font-size: 0.9em;
        }
        .ip-type {
            display: inline-block;
            padding: 3px 6px;
            background-color: #e74c3c;
            color: white;
            border-radius: 3px;
            font-size: 0.8em;
            margin-left: 5px;
        }
        .ip-type.ipv4 {
            background-color: #2ecc71;
        }
        .timestamp {
            text-align: right;
            color: #95a5a6;
            font-size: 0.8em;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>IP Address Change Detected</h1>
        <p>The following IP addresses have been detected:</p>
`)

	// Add IP information
	for _, ipInfo := range newIPs {
		htmlContent.WriteString("<div class=\"ip-item\">")
		htmlContent.WriteString("<div class=\"ip-address\">")
		htmlContent.WriteString(ipInfo.IPAddress)

		ipTypeClass := ""
		if ipInfo.Type == "IPv4" {
			ipTypeClass = " ipv4"
		}

		htmlContent.WriteString("<span class=\"ip-type")
		htmlContent.WriteString(ipTypeClass)
		htmlContent.WriteString("\">")
		htmlContent.WriteString(ipInfo.Type)
		htmlContent.WriteString("</span>")
		htmlContent.WriteString("</div>")

		htmlContent.WriteString("<div class=\"interface-name\">Interface: ")
		htmlContent.WriteString(ipInfo.InterfaceName)
		htmlContent.WriteString("</div>")
		htmlContent.WriteString("</div>")
	}

	// Add timestamp
	htmlContent.WriteString("<div class=\"timestamp\">Detected at: ")
	htmlContent.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	htmlContent.WriteString("</div>")

	htmlContent.WriteString(`
    </div>
</body>
</html>
`)

	// Prepare email headers and body
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      "IP Address Change Detected",
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(htmlContent.String())

	// Send email with retry mechanism
	auth := smtp.PlainAuth("", from, password, smtpHost)
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		err := smtp.SendMail(
			smtpHost+":"+smtpPort,
			auth,
			from,
			[]string{to},
			[]byte(message.String()),
		)

		if err != nil {
			fmt.Printf("Attempt %d: Error sending email: %v\n", i+1, err)
			if i < maxRetries-1 {
				time.Sleep(2 * time.Second) // Wait before retrying
				continue
			}
		} else {
			fmt.Println("Email sent successfully")
			break
		}
	}
}

// Web server for configuration
func startWebServer() {

	// Serve static files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Simple HTML form
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>IP Radar Configuration</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background-color: white;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        h1 {
            color: #2c3e50;
            border-bottom: 1px solid #eee;
            padding-bottom: 10px;
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input[type="text"], input[type="password"] {
            width: 100%%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        button {
            background-color: #3498db;
            color: white;
            border: none;
            padding: 10px 15px;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background-color: #2980b9;
        }
        .current-ips {
            margin-top: 20px;
            background-color: #f8f9fa;
            padding: 15px;
            border-radius: 4px;
        }
        .ip-item {
            margin-bottom: 10px;
            padding: 10px;
            background-color: #fff;
            border-left: 3px solid #3498db;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>IP Radar Configuration</h1>
        
        <form method="post" action="/save">
            <div class="form-group">
                <label for="from">Sender Email:</label>
                <input type="text" id="from" name="from" value="%s">
            </div>
            
            <div class="form-group">
                <label for="password">Email Password:</label>
                <input type="password" id="password" name="password" value="%s">
            </div>
            
            <div class="form-group">
                <label for="to">Recipient Email:</label>
                <input type="text" id="to" name="to" value="%s">
            </div>
            
            <div class="form-group">
                <label for="smtpHost">SMTP Host:</label>
                <input type="text" id="smtpHost" name="smtpHost" value="%s">
            </div>
            
            <div class="form-group">
                <label for="smtpPort">SMTP Port:</label>
                <input type="text" id="smtpPort" name="smtpPort" value="%s">
            </div>
            
            <button type="submit">Save Configuration</button>
        </form>
        
        <div class="current-ips">
            <h2>Current IP Addresses</h2>
            %s
        </div>
    </div>
</body>
</html>
`, config.Email.From, config.Email.Password, config.Email.To, config.Email.SmtpHost, config.Email.SmtpPort, getCurrentIPsHTML())

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Handle form submission
	http.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		// Update configuration
		config.Email.From = r.FormValue("from")
		config.Email.Password = r.FormValue("password")
		config.Email.To = r.FormValue("to")
		config.Email.SmtpHost = r.FormValue("smtpHost")
		config.Email.SmtpPort = r.FormValue("smtpPort")

		// Save to file
		err = saveConfig("config.json")
		if err != nil {
			http.Error(w, "Error saving configuration", http.StatusInternalServerError)
			return
		}

		// Redirect back to the main page
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Start the server
	log.Fatal(http.ListenAndServe(":8087", nil))
}

func getCurrentIPsHTML() string {
	ips := getRealIPs()
	if len(ips) == 0 {
		return "<p>No IP addresses detected.</p>"
	}

	var html strings.Builder
	for _, ip := range ips {
		html.WriteString(fmt.Sprintf(`
            <div class="ip-item">
                <strong>%s</strong> (%s)<br>
                Interface: %s
            </div>
        `, ip.IPAddress, ip.Type, ip.InterfaceName))
	}

	return html.String()
}

func ensureConfigExists(filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		defaultConfig := `{
  "email": {
    "from": "Sender's email address",
    "password": "Sender message key",
    "to": "The recipient's email address",
    "smtpHost": "smtp.126.com  Modify it for your mailbox provider",
    "smtpPort": "Modify it for your mailbox provider"
  }
}`
		err := os.WriteFile(filename, []byte(defaultConfig), 0644)
		if err != nil {
			log.Fatalf("Error creating default config.json: %v", err)
		}
		fmt.Println("Default config.json created.")
	}
}
