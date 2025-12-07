# Database Schema Design

## Users Table
- id: PRIMARY KEY
- username: VARCHAR(255) UNIQUE
- email: VARCHAR(255) UNIQUE
- created_at: TIMESTAMP

## Posts Table
- id: PRIMARY KEY
- user_id: FOREIGN KEY REFERENCES users(id)
- title: VARCHAR(500)
- content: TEXT
- published_at: TIMESTAMP

## Comments Table
- id: PRIMARY KEY
- post_id: FOREIGN KEY REFERENCES posts(id)
- user_id: FOREIGN KEY REFERENCES users(id)
- comment_text: TEXT
- created_at: TIMESTAMP

## Indexes
- CREATE INDEX idx_posts_user_id ON posts(user_id)
- CREATE INDEX idx_comments_post_id ON comments(post_id)
