import React from "react";
import request from "axios";
import { Spin, message, Icon } from "antd";

import { UPSTREAMS } from "../urls";
import "./director.sass";

function createList(data, key, name) {
  if (!data[key]) {
    return;
  }
  const value = data[key];
  let arr = null;
  if (Array.isArray(value)) {
    arr = value.map(item => {
      return <li key={item}>{item}</li>;
    });
  } else {
    const keys = Object.keys(value);
    arr = keys.map(item => {
      return <li key={item}>{`${item}:${value[item]}`}</li>;
    });
  }

  return (
    <div>
      <h5>{name || key}</h5>
      <ul>{arr}</ul>
    </div>
  );
}

class Director extends React.Component {
  state = {
    loading: false,
    shrinks: {},
    upstreams: null
  };
  renderUpstreams() {
    const { loading, upstreams, shrinks } = this.state;
    if (loading || !upstreams) {
      return;
    }
    if (upstreams.length === 0) {
      return (
        <div className="noUpstreams">
          <Icon type="info-circle" />
          There is no upstream.
        </div>
      );
    }
    return upstreams.map(item => {
      const { name, policy } = item;
      const shrinked = shrinks[name];

      let expandShrinke = null;
      const toggle = e => {
        e.preventDefault();
        const data = {};
        data[name] = !shrinks[name];
        this.setState({
          shrinks: Object.assign(shrinks, data)
        });
      };
      if (shrinked) {
        expandShrinke = (
          <a href="/expand" title="expand" onClick={e => toggle(e)}>
            <Icon type="arrows-alt" />
          </a>
        );
      } else {
        expandShrinke = (
          <a href="/shrink" title="shrink" onClick={e => toggle(e)}>
            <Icon type="shrink" />
          </a>
        );
      }
      const moreInfos = [];
      if (!shrinked) {
        const servers = item.servers.map(server => {
          let icon = <Icon className="status" type="check-circle" />;
          if (server.status !== "healthy") {
            icon = <Icon className="status sick" type="close-circle" />;
          }
          return (
            <li key={server.url}>
              {server.url}
              {icon}
              {server.backup && <span className="backup">backup</span>}
            </li>
          );
        });
        moreInfos.push(<h5>servers</h5>);
        moreInfos.push(<ul>{servers}</ul>);
        moreInfos.push(createList(item, "hosts"));
        moreInfos.push(createList(item, "prefixs"));
        moreInfos.push(createList(item, "rewrites"));
        moreInfos.push(createList(item, "header"));
        moreInfos.push(createList(item, "requestHeader", "request header"));
      }

      return (
        <div className="upstream" key={name}>
          <h4>
            <div className="functions">{expandShrinke}</div>
            {name}
            {policy && (
              <span className="policy" title="policy">
                {policy}
              </span>
            )}
          </h4>
          {moreInfos}
        </div>
      );
    });
  }
  render() {
    const { loading } = this.state;
    return (
      <div className="Director">
        {loading && (
          <div
            style={{
              textAlign: "center",
              paddingTop: "50px"
            }}
          >
            <Spin tip="Loading..." />
          </div>
        )}
        {this.renderUpstreams()}
      </div>
    );
  }
  async componentDidMount() {
    this.setState({
      loading: true
    });
    try {
      const { data } = await request.get(UPSTREAMS);
      this.setState({
        upstreams: data.upstreams
      });
    } catch (err) {
      message.error(err.message);
    } finally {
      this.setState({
        loading: false
      });
    }
  }
}

export default Director;
