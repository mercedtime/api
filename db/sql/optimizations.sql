CREATE INDEX course_type_idx ON course (type);

CREATE INDEX aux_course_crn_idx ON aux (course_crn);
CREATE INDEX non_zero_aux_course_crn_idx
          ON aux (course_crn) WHERE course_crn != 0;

-- Holy bageebus this thing is fast
--
-- This view is actually being used in application logic,
-- most of the other views are just for managment.
CREATE MATERIALIZED
VIEW catalog AS
     SELECT c.*, array_to_json(sub) AS subcourses
       FROM course c
 LEFT OUTER JOIN (
     SELECT array_agg(json_build_object(
		 	'id', 				 course.id,
		    'crn',               aux.crn,
		    'course_crn',        aux.course_crn,
		    'section',           aux.section,
			'type',              course.type,
		    'days',              course.days,
		    'enrolled',          course.enrolled,
		    'start_time',        aux.start_time,
		    'end_time',          aux.end_time,
		    'building_room',     aux.building_room,
		    'instructor_id',     aux.instructor_id,
		    'updated_at',        aux.updated_at,
		    'course_updated_at', course.updated_at
      )) AS sub, course_crn
       FROM aux
       JOIN course ON aux.crn = course.crn
      WHERE aux.course_crn != 0
   GROUP BY aux.course_crn
) a ON c.crn = a.course_crn;

CREATE UNIQUE INDEX ON catalog (id);
-- Run this every once in a while
--REFRESH MATERIALIZED VIEW CONCURRENTLY catalog;
