CREATE TABLE term (
    id INT,
    name VARCHAR(6)
);

INSERT INTO term (
    id,
    name
) VALUES
    (1, 'spring'),
    (2, 'summer'),
    (3, 'fall');

CREATE TABLE subject (
    code    VARCHAR(4) UNIQUE NOT NULL,
    name    TEXT,
    year    INTEGER NOT NULL,
    term_id INT NOT NULL
);

INSERT INTO subject (
    code,
    name,
    year,
    term_id
) VALUES
    ('ANTH', 'Anthropology', 2021, 1),
    ('BIO', 'Biological Sciences', 2021, 1),
    ('BIOE', 'Bioengineering', 2021, 1),
    ('CCST', 'Chicano Chicana Studies', 2021, 1),
    ('CHEM', 'Chemistry', 2021, 1),
    ('CHN', 'Chinese', 2021, 1),
    ('COGS', 'Cognitive Science', 2021, 1),
    ('CRES', 'Critical Race and Ethnic Studies', 2021, 1),
    ('CRS', 'Community Research and Service', 2021, 1),
    ('CSE', 'Computer Science and Engineering', 2021, 1),
    ('ECON', 'Economics', 2021, 1),
    ('EECS', 'Electrical Engineering and Computer Science', 2021, 1),
    ('ENG', 'English', 2021, 1),
    ('ENGR', 'Engineering', 2021, 1),
    ('ENVE', 'Environmental Engineering', 2021, 1),
    ('ES', 'Environmental Systems (GR)', 2021, 1),
    ('ESS', 'Environmental Systems Science', 2021, 1),
    ('FRE', 'French', 2021, 1),
    ('GASP', 'Global Arts Studies Program', 2021, 1),
    ('HIST', 'History', 2021, 1),
    ('HS', 'Heritage Studies', 2021, 1),
    ('IH', 'Interdisciplinary Humanities', 2021, 1),
    ('JPN', 'Japanese', 2021, 1),
    ('MATH', 'Mathematics', 2021, 1),
    ('MBSE', 'Materials and BioMat Sci & Engr', 2021, 1),
    ('ME', 'Mechanical Engineering', 2021, 1),
    ('MGMT', 'Management', 2021, 1),
    ('MIST', 'Management of Innovation, Sustainability and Technology', 2021, 1),
    ('MSE', 'Materials Science and Engineering', 2021, 1),
    ('NSED', 'Natural Sciences Education', 2021, 1),
    ('PH', 'Public Health', 2021, 1),
    ('PHIL', 'Philosophy', 2021, 1),
    ('PHYS', 'Physics', 2021, 1),
    ('POLI', 'Political Science', 2021, 1),
    ('PSY', 'Psychology', 2021, 1),
    ('QSB', 'Quantitative and Systems Biology', 2021, 1),
    ('SOC', 'Sociology', 2021, 1),
    ('SPAN', 'Spanish', 2021, 1),
    ('SPRK', 'Spark', 2021, 1),
    ('WRI', 'Writing', 2021, 1);
