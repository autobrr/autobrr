-- keys created before scopes were enforced had full access with empty scopes
UPDATE api_key
SET scopes = ARRAY ['*']
WHERE cardinality(scopes) = 0;
