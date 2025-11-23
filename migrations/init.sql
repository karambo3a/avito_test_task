CREATE TABLE IF NOT EXISTS team (
    team_name VARCHAR(255) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,

    FOREIGN KEY (team_name) REFERENCES team(team_name)
);

CREATE INDEX is_active_team_idx ON users(team_name, is_active);
CREATE INDEX is_active_user_idx ON users(user_id, is_active);

CREATE TABLE IF NOT EXISTS pr (
    pr_id VARCHAR(255) PRIMARY KEY,
    pr_name VARCHAR(255) NOT NULL,
    author_id VARCHAR(255) NOT NULL,
    status VARCHAR(6) DEFAULT 'OPEN',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    merged_at TIMESTAMP DEFAULT NULL,

    FOREIGN KEY (author_id) REFERENCES users(user_id)
);

CREATE INDEX pr_author_id_idx ON pr(author_id);
CREATE INDEX pr_status_idx ON pr(status);

CREATE TABLE IF NOT EXISTS reviewer_x_pr (
    user_id VARCHAR(255),
    pr_id VARCHAR(255),

    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (pr_id) REFERENCES pr(pr_id),

    PRIMARY KEY (user_id, pr_id)
);

CREATE INDEX reviewer_x_pr_pr_id_idx ON reviewer_x_pr(pr_id);
