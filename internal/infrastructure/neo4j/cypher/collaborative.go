package cypher

const Collaborative = `
MATCH (u:User {id: $userId})-[:WATCHED]->(m:Movie)<-[:WATCHED]-(similar:User)
WHERE similar.id <> $userId
WITH similar, COUNT(m) AS overlap ORDER BY overlap DESC LIMIT 20
MATCH (similar)-[:LIKED]->(candidate:Movie)
WHERE NOT (u)-[:WATCHED]->(candidate)
  AND candidate.imdbRating >= $minRating
WITH candidate, COUNT(similar) AS score
RETURN candidate ORDER BY score DESC
LIMIT $limit
`
