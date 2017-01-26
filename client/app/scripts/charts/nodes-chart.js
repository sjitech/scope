import debug from 'debug';
import React from 'react';
import { connect } from 'react-redux';
import { assign, pick, includes } from 'lodash';
import { Map as makeMap, fromJS } from 'immutable';
import { createSelector } from 'reselect';
import timely from 'timely';

import { event as d3Event, select } from 'd3-selection';
import { zoom, zoomIdentity } from 'd3-zoom';

import { nodeAdjacenciesSelector, adjacentNodesSelector } from '../selectors/chartSelectors';
import { clickBackground } from '../actions/app-actions';
import Logo from '../components/logo';
import { doLayout } from './nodes-layout';
import NodesChartElements from './nodes-chart-elements';
import { getActiveTopologyOptions } from '../utils/topology-utils';

import { restoredZoomState } from '../selectors/nodes-chart-zoom';
import { selectedNodeInFocus } from '../selectors/nodes-chart-focus';
import { updatedLayout, restoredLayout } from '../selectors/nodes-chart-layout';

const log = debug('scope:nodes-chart');
const ZOOM_CACHE_FIELDS = ['scale', 'panTranslateX', 'panTranslateY', 'minScale', 'maxScale'];


class NodesChart extends React.Component {
  constructor(props, context) {
    super(props, context);

    this.state = {
      layoutNodes: makeMap(),
      layoutEdges: makeMap(),
      scale: 0,
      minScale: 0,
      maxScale: 0,
      panTranslateX: 0,
      panTranslateY: 0,
      selectedScale: 1,
      height: props.height || 0,
      width: props.width || 0,
      zoomCache: {},
    };

    this.handleMouseClick = this.handleMouseClick.bind(this);
    this.zoomed = this.zoomed.bind(this);
  }

  componentWillMount() {
    this.setState(updatedLayout(this.state, this.props));
  }

  // shouldComponentUpdate(nextProps) {
  //   return !(this.props.forceRelayout && !nextProps.forceRelayout);
  // }

  componentWillReceiveProps(nextProps) {
    // gather state, setState should be called only once here
    const state = assign({}, this.state);

    // const topologyChanged = nextProps.topologyId !== this.props.topologyId;

    // wipe node states when showing different topology
    // if (topologyChanged) {
    //   assign(state, emptyLayoutState);
    // }

    if (nextProps.topologyId !== this.props.topologyId ||
      this.props.selectedNodeId !== nextProps.selectedNodeId) {
      // saving previous zoom state
      const prevZoom = pick(state, ZOOM_CACHE_FIELDS);
      const zoomCache = assign({}, state.zoomCache);
      zoomCache[this.props.topologyId] = prevZoom;
      assign(state, { zoomCache });
      // console.log('Cached zoom', prevZoom);
    }

    // reset layout dimensions only when forced
    state.height = nextProps.forceRelayout ? nextProps.height : (state.height || nextProps.height);
    state.width = nextProps.forceRelayout ? nextProps.width : (state.width || nextProps.width);

    // if (nextProps.forceRelayout || nextProps.nodes !== this.props.nodes) {
    assign(state, updatedLayout(state, nextProps));
    // }

    // if (nextProps.nodes.size > 0) {
    assign(state, restoredZoomState(state, nextProps));
    // }

    if (this.props.selectedNodeId !== nextProps.selectedNodeId) {
      assign(state, restoredLayout(state));
    }

    if (nextProps.selectedNodeId) {
      assign(state, selectedNodeInFocus(nextProps, state));
    }

    this.setZoom(state);
    this.setState(state);
  }

  componentDidMount() {
    // distinguish pan/zoom from click
    this.isZooming = false;
    this.zoom = zoom().on('zoom', this.zoomed);

    this.svg = select('.nodes-chart svg');
    this.svg.call(this.zoom);
  }

  componentWillUnmount() {
    // undoing .call(zoom)
    this.svg
      .on('mousedown.zoom', null)
      .on('onwheel', null)
      .on('onmousewheel', null)
      .on('dblclick.zoom', null)
      .on('touchstart.zoom', null);
  }

  isSmallTopology() {
    return this.state.layoutNodes.size < 100;
  }

  render() {
    const { panTranslateX, panTranslateY, scale, selectedScale } = this.state;

    // not passing translates into child components for perf reasons
    const translate = [panTranslateX, panTranslateY];
    const transform = `translate(${translate}) scale(${scale})`;
    const svgClassNames = this.props.isEmpty ? 'hide' : '';
    const isAnimated = this.isSmallTopology();

    return (
      <div className="nodes-chart">
        <svg
          width="100%" height="100%" id="nodes-chart-canvas"
          className={svgClassNames} onClick={this.handleMouseClick}>
          <g transform="translate(24,24) scale(0.25)">
            <Logo />
          </g>
          <NodesChartElements
            layoutNodes={this.state.layoutNodes}
            layoutEdges={this.state.layoutEdges}
            focusMagnifyFactor={selectedScale / scale}
            transform={transform}
            isAnimated={isAnimated} />
        </svg>
      </div>
    );
  }

  handleMouseClick() {
    if (!this.isZooming || this.props.selectedNodeId) {
      this.props.clickBackground();
    } else {
      this.isZooming = false;
    }
  }

  zoomed() {
    this.isZooming = true;
    // don't pan while node is selected
    if (!this.props.selectedNodeId) {
      this.setState({
        panTranslateX: d3Event.transform.x,
        panTranslateY: d3Event.transform.y,
        scale: d3Event.transform.k
      });
    }
  }

  setZoom(newZoom) {
    this.zoom = this.zoom.scaleExtent([newZoom.minScale, newZoom.maxScale]);
    this.svg.call(this.zoom.transform, zoomIdentity
      .translate(newZoom.panTranslateX, newZoom.panTranslateY)
      .scale(newZoom.scale));
  }
}


function mapStateToProps(state) {
  return {
    nodes: nodeAdjacenciesSelector(state),
    adjacentNodes: adjacentNodesSelector(state),
    forceRelayout: state.get('forceRelayout'),
    selectedNodeId: state.get('selectedNodeId'),
    topologyId: state.get('currentTopologyId'),
    topologyOptions: getActiveTopologyOptions(state)
  };
}


export default connect(
  mapStateToProps,
  { clickBackground }
)(NodesChart);
