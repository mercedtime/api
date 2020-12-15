CREATE TABLE term (
    id INT,
    name VARCHAR(6)
);

INSERT INTO term (id, name) VALUES (1, 'spring');
INSERT INTO term (id, name) VALUES (2, 'summer');
INSERT INTO term (id, name) VALUES (3, 'fall');

CREATE TABLE subject (
    code    VARCHAR(4) UNIQUE NOT NULL,
    name    TEXT,
    year    INTEGER NOT NULL,
    term_id INT NOT NULL
);

INSERT INTO subject (code, name, year, term_id) VALUES ('ANTH', 'Anthropology', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('BIO', 'Biological Sciences', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('BIOE', 'Bioengineering', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('CCST', 'Chicano Chicana Studies', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('CHEM', 'Chemistry', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('CHN', 'Chinese', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('COGS', 'Cognitive Science', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('CRES', 'Critical Race and Ethnic Studies', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('CRS', 'Community Research and Service', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('CSE', 'Computer Science and Engineering', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('ECON', 'Economics', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('EECS', 'Electrical Engineering and Computer Science', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('ENG', 'English', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('ENGR', 'Engineering', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('ENVE', 'Environmental Engineering', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('ES', 'Environmental Systems (GR)', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('ESS', 'Environmental Systems Science', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('FRE', 'French', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('GASP', 'Global Arts Studies Program', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('HIST', 'History', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('HS', 'Heritage Studies', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('IH', 'Interdisciplinary Humanities', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('JPN', 'Japanese', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('MATH', 'Mathematics', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('MBSE', 'Materials and BioMat Sci & Engr', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('ME', 'Mechanical Engineering', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('MGMT', 'Management', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('MIST', 'Management of Innovation, Sustainability and Technology', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('MSE', 'Materials Science and Engineering', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('NSED', 'Natural Sciences Education', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('PH', 'Public Health', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('PHIL', 'Philosophy', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('PHYS', 'Physics', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('POLI', 'Political Science', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('PSY', 'Psychology', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('QSB', 'Quantitative and Systems Biology', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('SOC', 'Sociology', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('SPAN', 'Spanish', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('SPRK', 'Spark', 2021, 1);
INSERT INTO subject (code, name, year, term_id) VALUES ('WRI', 'Writing', 2021, 1);
