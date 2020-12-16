# MercedTime's REST API

| method     | endpoint                            | description                         | protected |
| ------     | --------                            | -----------                         | --------- |
| **GET**    | [`/lectures`](#list-lectures)       | List lectures                       | ❌        |
| **GET**    | [`/labs`](#list-labs)               | List labs                           | ❌        |
| **GET**    | [`/exams`](#list-exams)             | List exams                          | ❌        |
| **GET**    | [`/discussions`](#list-discussions) | List discussions                    | ❌        |
| **GET**    | [`/instructors`](#list-instructors) | List instructors                    | ❌        |
| **GET**    | `/lecture/:crn`                     | Get a lecture                       | ❌        |
| **DELETE** | `/lecture/:crn`                     | Delete a lecture                    | ✔️         |
| **GET**    | `/lecture/:crn/exam`                | Get a lecture's exam                | ❌        |
| **GET**    | `/lecture/:crn/labs`                | Get a lecture's lab sections        | ❌        |
| **GET**    | `/lecture/:crn/instructor`          | Get a lecture's list of instructors | ❌        |
| **GET**    | `/user/:id`                         | Get a user                          | ✔️         |
| **POST**   | `/user`                             | Create a user                       | ✔️         |
| **DELETE** | `/user/:id`                         | Delete a user                       | ✔️         |
| **POST**   | `/login`                            | Get login credentials               | ❌        |

# TODO: Coding Shit

- PUT /user For updating a user
- GET /subject Get a subject code, description, and id
- GET /refresh For getting a refresh token
- Add a sign in with email option (probably just changing the request body for /login).
- GET /lecture/:crn/enrollment For getting the historic enrollment stats
- GET /standalone
- Add a "last notified" field to the user table. If we want to do notifications in the future we will probably need to do date comparisons with recently updated courses
- To control which year and term for which the data is returned, write a "State" struct that contains this global route state to be accessed globally be the api. (maybe make a new internal package routes with "routes.State")

# TODO: Thinking Shit

- What is the best way to expose enrollment data in the api?
- Need to build a prerequisites tree.
- The instructor id system is a horrible hack, use a SERIAL type instead
- To make this useful for different terms, we cannot rely on crn as our only primary key, need to start making my own course ids

---

**POST** `/login`
<a name="login"></a>

Get access to protected resources by giving the login endpoint your credentials.

Responses with a [JSON Web Token (_JWT_)](https://jwt.io/)

<details>
  <summary>Example Request Body</summary>

```json
{
    "username": "my username",
    "password": "*R(Py*(P*F$JIjF:EJ"
}
```

</details><br>

**GET** `/lectures`
<a name="list-lectures"></a>

- `limit=<limit>` __int__ Limit the number of results to `<limit>`
- `offset=<offset>` __int__ Offset the response list by some offset number
- `subject=<code>` __string__ Only return the lectures for the subject that matches `<code>`

<details>
<summary>Example Response</summary>

```json
{}
```

</details><br>

**GET** `/labs`
<a name="list-labs"></a>

- `limit=<limit>` __int__ Limit the number of results to `<limit>`
- `offset=<offset>` __int__ Offset the response list by some offset number

<details>
<summary>Example Response</summary>

```json
{}
```

</details><br>

**GET** `/exams`
<a name="list-exams"></a>

- `limit=<limit>` __int__ Limit the number of results to `<limit>`
- `offset=<offset>` __int__ Offset the response list by some offset number

<details>
<summary>Example Response</summary>

```json
{}
```

</details><br>

**GET** `/discussions`
<a name="list-discussions"></a>

- `limit=<limit>` __int__ Limit the number of results to `<limit>`
- `offset=<offset>` __int__ Offset the response list by some offset number

<details>
<summary>Example Response</summary>

```json
{}
```

</details><br>

**GET** `/instructors`
<a name="list-instructors"></a>

- `limit=<limit>` __int__ Limit the number of results to `<limit>`
- `offset=<offset>` __int__ Offset the response list by some offset number

<details>
<summary>Example Response</summary>

```json
{}
```

</details><br>

---

### Errors

Most of the endpoints will respond with the same error type that looks something like this.

```json
{
    "error": "you did a bad thing",
    "status": 400
}
```
