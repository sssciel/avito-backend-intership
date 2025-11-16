-- sql
DROP TABLE IF EXISTS pull_request_reviewers;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS pull_requests;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
                       user_id VARCHAR(255) PRIMARY KEY,
                       username VARCHAR(255) NOT NULL,
                       is_active BOOLEAN NOT NULL DEFAULT true,
                       created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE teams (
                       id SERIAL PRIMARY KEY,
                       name VARCHAR(255) NOT NULL UNIQUE,
                       created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE pull_requests (
                               id SERIAL PRIMARY KEY,
                               pull_request_id VARCHAR(255) NOT NULL UNIQUE,
                               name VARCHAR(255) NOT NULL,
                               author_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
                               status VARCHAR(50) NOT NULL,
                               merged_at TIMESTAMP,
                               created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE team_members (
                              team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
                              user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
                              joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
                              PRIMARY KEY (team_id, user_id)
);

CREATE TABLE pull_request_reviewers (
                                        pull_request_id INTEGER NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
                                        reviewer_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
                                        assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
                                        PRIMARY KEY (pull_request_id, reviewer_id)
);

CREATE INDEX idx_pull_requests_author ON pull_requests(author_id);
CREATE INDEX idx_team_members_team ON team_members(team_id);
CREATE INDEX idx_pr_reviewers_reviewer ON pull_request_reviewers(reviewer_id);