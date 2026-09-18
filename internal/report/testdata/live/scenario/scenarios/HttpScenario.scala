package org.galaxio.performance.live.scenarios

import io.gatling.core.Predef._
import io.gatling.core.structure.ScenarioBuilder
import org.galaxio.performance.live.cases._

object HttpScenario {
  def apply(): ScenarioBuilder = new HttpScenario().scn
}

class HttpScenario {

  val scn: ScenarioBuilder = scenario("Http Scenario")
    .exec(HttpActions.getMainPage)
    .doIf(session => session.userId % 100 < 7)(exec(HttpActions.getReport))
    .doIf(session => session.userId % 100 >= 97)(exec(HttpActions.getExport))

}
