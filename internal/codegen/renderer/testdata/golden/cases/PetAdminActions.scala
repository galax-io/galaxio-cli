package org.galaxio.performance.cases

import io.gatling.http.Predef._
import io.gatling.core.Predef._

object PetAdminActions {
  val createPet = http("POST /pets/{petId}")
    .post("/api/v1/pets/${petId}")
    .queryParam("trace-id", "${traceId}")
    .header("Authorization", "${authorization}")
    .body(ElFileBody("bodies/createPet.json")).asJson
    .check(status is 201)
}
