# MercedTime's REST API

| method     | endpoint                   | description                         |
| ------     | --------                   | -----------                         |
| **GET**    | `/lectures`                | List lectures                       |
| **GET**    | `/labs`                    | List labs                           |
| **GET**    | `/exams`                   | List exams                          |
| **GET**    | `/discussions`             | List discussions                    |
| **GET**    | `/instructors`             | List instructors                    |
| **GET**    | `/lecture/:crn`            | Get a lecture                       |
| **DELETE** | `/lecture/:crn`            | Delete a lecture                    |
| **GET**    | `/lecture/:crn/exam`       | Get a lecture's exam                |
| **GET**    | `/lecture/:crn/labs`       | Get a lecture's lab sections        |
| **GET**    | `/lecture/:crn/instructor` | Get a lecture's list of instructors |
| **GET**    | `/user/:id`                | Get a user                          |
| **POST**   | `/user`                    | Create a user                       |
| **DELETE** | `/user/:id`                | Delete a user                       |
| **POST**   | `/login`                   | Get login credentials               |

# TODO: API
- PUT /user
- GET /subject Get a subject code, description, and id

# TODO: Thinking Shit
- What is the best way to give enrollment data in the api?
- Need to build a prerequisites tree.

