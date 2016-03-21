///<reference path="../../../headers/common.d.ts" />

import angular from 'angular';
import _ from 'lodash';

export class InfluxConfigCtrl {
  static templateUrl = 'partials/config.html';

  current: any;

  influxVersions = [
    { name: "< 0.10.0", value: 1},
    { name: "> 0.10.0", value: 2}
  ];

  /** @ngInject **/
  constructor($scope) {

    this.current.jsonData.influxVersion = this.current.jsonData.influxVersion || 1;
  }
}

