package cypher

const Popular = `
MATCH (m:Movie)-[:HAS_GENRE]->(g:Genre)<-[:INTERESTED_IN]-(u:User {id: $userId})
WHERE NOT (u)-[:WATCHED]->(m)
  AND m.imdbRating >= $minRating
RETURN m ORDER BY m.imdbRating DESC
LIMIT $limit
`
