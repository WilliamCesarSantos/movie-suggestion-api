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
MERGE (u:User {email: 'william_cesar_santos@hotmail.com'})
ON CREATE SET
  u.id       = randomUUID(),
  u.name     = 'William',
  u.password = '$argon2id$v=19$m=65536,t=3,p=4$qNkRswLidbmSiP0zbdj81g$Y6hkfgo8OaAMoGT0hQLUlfFVWGjH2V2Tsv28qA2M0j4',
  u.roles    = ['*'],
  u.createdAt = datetime();
