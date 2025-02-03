# BW4E Systems Discord Bot (Go Rewrite)

This is the official Discord bot for BW4E Systems, rewritten in Go for improved performance and resource efficiency. The bot handles email verification, role assignment, and channel management for The BW4E Discord server.

---

## Features
- **Email Verification**: Validates user emails against a Google Sheets database.
- **Role Assignment**: Automatically assigns roles based on email verification results.
- **Slash Commands**:
  - `/hide`: Hides a specified channel for the user.
  - `/unhide`: Unhides a specified channel for the user.

---

## Setup Instructions

### Prerequisites
1. **Go**: Ensure Go is installed on your system. [Download Go](https://go.dev/dl/).
2. **Google Cloud Project**:
   - Enable the Google Sheets API.
   - Create a service account and download the credentials JSON file.

### Steps
1. Clone this repository:

git clone https://github.com/YOUR_USERNAME/bw4e-bot.git
cd bw4e-bot


2. Create your `config.json` file:
- Copy `example-config.json` to `config.json` and fill in the required fields:
  ```
  {
      "discordToken": "YOUR_DISCORD_BOT_TOKEN",
      "guildId": "YOUR_GUILD_ID",
      "emailChannelId": "YOUR_EMAIL_CHANNEL_ID",
      "roleFoundId": "ROLE_ID_FOR_VERIFIED_USERS",
      "roleNotFoundId": "ROLE_ID_FOR_UNVERIFIED_USERS",
      "credentialsPath": "./credentials.json",
      "spreadsheetId": "YOUR_GOOGLE_SHEETS_SPREADSHEET_ID"
  }
  ```

3. Add your Google service account credentials:
- Copy `example-credentials.json` to `credentials.json` and replace placeholder values with your service account details.

4. Build and run the bot:

go build -o bw4e-bot
./bw4e-bot



---

## Example Google Sheets Table

| **Email**           |
|----------------------|
| user1@example.com    |
| user2@example.com    |
| testuser@mail.com    |

- Column A (`Email`) contains the list of valid email addresses for verification.

---

## License
This project is **closed-source** and proprietary to BW4E Systems. Redistribution, modification, or use of this code outside of BW4E Systems is strictly prohibited without express written permission.

