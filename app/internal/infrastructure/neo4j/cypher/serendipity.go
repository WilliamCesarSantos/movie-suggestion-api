package cypher

const Serendipity = `
MATCH (u:User {id: $userId})-[:WATCHED]->(watched:Movie)-[:HAS_GENRE]->(g:Genre)
WITH u, COLLECT(DISTINCT g.name) AS knownGenres
MATCH (candidate:Movie)-[:HAS_GENRE]->(cg:Genre)
WHERE NOT (u)-[:WATCHED]->(candidate)
  AND NOT cg.name IN knownGenres
  AND candidate.imdbRating >= $serendipityMinRating
RETURN candidate ORDER BY candidate.imdbRating DESC
LIMIT $limit
`
