package org.galaxio.performance.scenarios

import io.gatling.core.Predef._
import io.gatling.core.structure.ScenarioBuilder
import org.galaxio.performance.cases._

object PetAdminScenario {
  def apply(): ScenarioBuilder = new PetAdminScenario().scn
}

class PetAdminScenario {

  val scn: ScenarioBuilder = scenario("PetAdmin Scenario")
    .exec(PetAdminActions.createPet)

}
