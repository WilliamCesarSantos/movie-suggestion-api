package cypher

const Popular = `
MATCH (u:User {id: $userId})
MATCH (m:Movie)
WHERE NOT (u)-[:WATCHED]->(m)
  AND m.imdbRating >= $minRating
RETURN m ORDER BY m.imdbRating DESC
LIMIT $limit
`
