# TeleChat - Real-Time Chat Application

**Author:** Hamza Abdul Karim

A modern, sleek real-time chat application built with Go and WebSockets, featuring a beautiful UI and comprehensive chat functionality.

## 🚀 Features

- Real-time messaging with WebSocket connections
- User authentication with custom usernames
- Online user list with live count
- Typing indicators to see who's typing in real-time
- Message editing and deletion for your own messages
- Connection status with visual feedback
- Auto-reconnection on connection loss
- Responsive design for desktop and mobile
- Modern UI with animations and gradients

## 🛠️ Technology Stack

- Backend: Go with gorilla/websocket
- Frontend: HTML5, CSS3, JavaScript (ES6+)
- WebSocket: Real-time bidirectional communication
- Concurrency: Go goroutines for handling multiple clients
- UI: Modern CSS with animations and responsive design

## 📦 Installation & Setup

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd TeleChat-master
   ```

2. Install Go dependencies:
   ```bash
   go mod tidy
   ```

3. Run the server:
   ```bash
   go run main.go
   ```

4. Open your browser and navigate to:
   ```
   http://localhost:8080
   ```

## 🎯 How to Use

1. Enter your desired username on the site
2. Start chatting by typing messages and pressing Enter or clicking send
3. View online users in the sidebar
4. Edit your messages by hovering and clicking the edit icon
5. Delete your messages by hovering and clicking the delete icon
6. See typing indicators for other users
7. Monitor your connection status in the top-right corner

## 🏗️ Project Structure

```
TeleChat-master/
├── main.go              # Go server with WebSocket handling
├── go.mod               # Go module dependencies
├── static/              # Frontend assets
│   ├── index.html       # Main HTML structure
│   ├── style.css        # Modern CSS styling
│   └── script.js        # JavaScript WebSocket client
└── README.md            # This file
```

## 🔧 Server Architecture

- Hub: Central message broker managing all client connections
- Client: Represents each connected user with their WebSocket connection
- Message Types:
  - `message`: Regular chat messages
  - `typing`: Typing indicator updates
  - `edit`: Message editing
  - `delete`: Message deletion
  - `userList`: Online users update

## 🎨 UI Features

- Modern, clean design with gradient backgrounds
- Smooth animations and transitions
- Responsive layout for desktop, tablet, and mobile
- Dark/light accents with purple gradients
- Font Awesome icons for better UX
- Interactive hover effects

## 🔒 Security Features

- Server-side message validation
- Proper WebSocket origin checking (CORS)
- Automatic cleanup of dead connections
- Safe input sanitization

## 🌐 Browser Support

- Chrome 60+
- Firefox 55+
- Safari 12+
- Edge 79+

## 📱 Mobile Support

Fully responsive and works well on:
- Mobile phones (iOS/Android)
- Tablets (iPad/Android)
- Desktop browsers

## 🎉 Getting Started

1. Run the server: `go run main.go`
2. Open multiple browser tabs to `http://localhost:8080`
3. Enter different usernames in each tab
4. Start chatting and enjoy all features!

## 🤝 Contributing

Contributions are welcome! You can:
- Report bugs
- Suggest new features
- Submit pull requests
- Improve documentation

## 📄 License

This project is open source under the MIT License.

---

**Built with ❤️ using Go and modern web technologies**
