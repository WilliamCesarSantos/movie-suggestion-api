// Create constraints
CREATE CONSTRAINT movie_imdb_id IF NOT EXISTS FOR (m:Movie) REQUIRE m.imdbId IS UNIQUE;
CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;
CREATE CONSTRAINT user_email IF NOT EXISTS FOR (u:User) REQUIRE u.email IS UNIQUE;
CREATE CONSTRAINT genre_name IF NOT EXISTS FOR (g:Genre) REQUIRE g.name IS UNIQUE;
CREATE CONSTRAINT actor_imdb_id IF NOT EXISTS FOR (a:Actor) REQUIRE a.imdbId IS UNIQUE;
CREATE CONSTRAINT director_imdb_id IF NOT EXISTS FOR (d:Director) REQUIRE d.imdbId IS UNIQUE;

// Create indexes
CREATE INDEX movie_id IF NOT EXISTS FOR (m:Movie) ON (m.id);
CREATE INDEX movie_rating IF NOT EXISTS FOR (m:Movie) ON (m.imdbRating);
