-- +goose Up
ALTER TABLE event_settings ADD aptype VARCHAR(255); 
