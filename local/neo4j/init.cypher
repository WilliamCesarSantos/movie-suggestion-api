// Create constraints
CREATE CONSTRAINT movie_imdb_id IF NOT EXISTS FOR (m:Movie) REQUIRE m.imdbId IS UNIQUE;
CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;
CREATE CONSTRAINT user_email IF NOT EXISTS FOR (u:User) REQUIRE u.email IS UNIQUE;
CREATE CONSTRAINT genre_name IF NOT EXISTS FOR (g:Genre) REQUIRE g.name IS UNIQUE;
CREATE CONSTRAINT actor_name IF NOT EXISTS FOR (a:Actor) REQUIRE a.name IS UNIQUE;
CREATE CONSTRAINT director_name IF NOT EXISTS FOR (d:Director) REQUIRE d.name IS UNIQUE;

// Create indexes
CREATE INDEX movie_id IF NOT EXISTS FOR (m:Movie) ON (m.id);
CREATE INDEX movie_rating IF NOT EXISTS FOR (m:Movie) ON (m.imdbRating);

// Seed users
// The id must match local/postgresql/init.sql: auth (Postgres) and graph (Neo4j) share the user id.
// Properties mirror the graph user created by the API (neo4j.UserRepository.Create).
MERGE (u:User {email: 'william_cesar_santos@hotmail.com'})
ON CREATE SET
  u.id               = '519fbd7e-4973-4d08-a605-3f1fd44e2f26',
  u.name             = 'William',
  u.createdAt        = toString(datetime.truncate('second', datetime())),
  u.currentAlgorithm = 'POPULAR',
  u.watchCount       = 0,
  u.likeCount        = 0,
  u.dislikeCount     = 0;
