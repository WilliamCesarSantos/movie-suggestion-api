package cypher

const Collaborative = `
MATCH (u:User {id: $userId})-[:WATCHED]->(m:Movie)<-[:WATCHED]-(similar:User)
WHERE similar.id <> $userId
WITH u, similar, COUNT(m) AS overlap ORDER BY overlap DESC LIMIT 20
MATCH (similar)-[:LIKED]->(candidate:Movie)
WHERE NOT (u)-[:WATCHED]->(candidate)
  AND candidate.imdbRating >= $minRating
  AND ($title = '' OR toLower(candidate.title) CONTAINS toLower($title))
WITH candidate, COUNT(similar) AS score
RETURN candidate ORDER BY score DESC
LIMIT $limit
`
