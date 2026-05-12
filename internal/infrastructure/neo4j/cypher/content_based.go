package cypher

const ContentBased = `
MATCH (u:User {id: $userId})-[:LIKED]->(liked:Movie)
MATCH (liked)-[:HAS_GENRE|HAS_ACTOR|DIRECTED_BY]->(shared)<-[:HAS_GENRE|HAS_ACTOR|DIRECTED_BY]-(candidate:Movie)
WHERE NOT (u)-[:WATCHED]->(candidate)
  AND candidate.imdbRating >= $minRating
WITH candidate, COUNT(shared) AS score
RETURN candidate ORDER BY score DESC
LIMIT $limit
`
