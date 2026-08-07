# Solo Adventure Picker - Backend

Go backend with AI-powered adventure enhancement using Google Gemini.

## Setup

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Set up environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env and add your Gemini API key
   ```

3. **Get Gemini API Key:**
   - Visit [Google AI Studio](https://aistudio.google.com/app/apikey)
   - Create a new API key
   - Add it to your `.env` file

4. **Run locally:**
   ```bash
   go run scripts/seed.go   # first run only, seeds the SQLite database
   go run main.go
   ```
   No database server required — data is stored in a local SQLite file
   (`solo-adventure-picker.db` by default, override with `DB_PATH`).

5. **Or run with Docker:**
   ```bash
   docker-compose up --build
   ```

## AI Features

### Adventure Agent
- **Purpose**: Enhances adventure descriptions with personalized content
- **Model**: Gemini 2.0 Flash
- **Cost**: ~$0.001 per adventure enhancement
- **Fallback**: Gracefully falls back to static descriptions if AI fails

### API Enhancement
The `/random` endpoint now:
1. Fetches a random adventure from SQLite
2. Enhances the description using AI based on user preferences
3. Sets dynamic XP values based on adventure type
4. Returns personalized content

## Architecture

```
Routes → Adventure Agent → Gemini API
  ↓           ↓              ↓
SQLite → Enhanced Adventure → Response
```

## Future Features
- User-specific preferences storage
- Multiple agent types (coach, quest master)
- Advanced prompt engineering
- Cost optimization and caching 