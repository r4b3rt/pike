import React from 'react';
import { Route, HashRouter } from "react-router-dom";

import AppHeader from "./components/app_header";
import {
  CACHES_PATH,
  COMPRESSES_PATH,
  UPSTREAMS_PATH,
  LOCATIONS_PATH,
  SERVERS_PATH,
  ADMIN_PATH,
  HOME_PATH,
  INFLUXDB_PATH,
  CERT_PATH
} from "./paths";
import Caches from "./components/caches";
import Compresses from "./components/compress";
import Upstreams from "./components/upstreams";
import Locations from "./components/locations";
import Servers from "./components/servers";
import Admin from "./components/admin";
import Home from "./components/home";
import Cert from "./components/cert";
import Influxdb from "./components/influxdb";

function App() {
  return (
    <div className="App">
      <HashRouter>
        <AppHeader />
        <div>
          <Route path={CACHES_PATH} component={Caches} />
          <Route path={COMPRESSES_PATH} component={Compresses} />
          <Route path={UPSTREAMS_PATH} component={Upstreams} />
          <Route path={LOCATIONS_PATH} component={Locations} />
          <Route path={SERVERS_PATH} component={Servers} />
          <Route path={ADMIN_PATH} component={Admin} />
          <Route path={CERT_PATH} component={Cert} />
          <Route path={INFLUXDB_PATH} component={Influxdb} />
          <Route path={HOME_PATH} exact component={Home} />
        </div>
      </HashRouter>
    </div>
  );
}

export default App;
