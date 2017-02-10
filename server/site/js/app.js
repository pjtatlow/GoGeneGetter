var app = angular.module("TestApp",['ngSanitize', 'ngRoute', 'ngAnimate','ui.select','chart.js']);
app.config(['$routeProvider','$locationProvider',function($routeProvider, $locationProvider) {
    $routeProvider
    .when("/", {
        templateUrl : "/templates/home.html"
    })
    .when("/samples", {
        templateUrl : "/templates/filter_samples.html"
    })
    .when("/genes", {
        templateUrl : "/templates/filter_genes.html"
    })    
    .otherwise({ templateUrl : "/templates/404.html" });
    $locationProvider.html5Mode(true);
}]);

app.controller("MainController",['$scope','$http','$location','$rootScope',function($scope,$http,$location,$rootScope) {
    $scope.query = "";
    $scope.allMeta = null;
    $scope.meta = null;
    $scope.metaTypes = null;
    $scope.genes = {"results":[],"selected":[]};
    $scope.selectedValues = {};
    $scope.totalNumSamples = null;
    $scope.totalNumGenes = null;
    $scope.numSamples = 0;
    $scope.numCalculating = 0;

    $scope.btnText = "";
    $scope.targetRoute = "";

    $scope.metaType = {
        "current": null,
        "values": [],
        "selected":[]
    };

    $scope.labels = [];
    $scope.values = [];

    $scope.$watch("selectedValues", function(newValue,oldValue) {
        angular.forEach(newValue,function(value,key) {
            if (value.length == 0) {
                delete this[key]
            }
        },newValue)
        
        $scope.fetchMeta();
    }, true);

    $scope.fetchMeta = function() {
        $http({
            method: 'GET',
            url: '/api/meta?query='+JSON.stringify($scope.selectedValues)
        }).then(function successCallback(response) {
            console.log(response.data);
            $scope.meta = response.data;

            if ($scope.allMeta === null) {
                $scope.allMeta = angular.copy($scope.meta);
            }

            $scope.calcNumSamples();

            $scope.buildChartData(false);
        }, function errorCallback(response) {
            console.error(response)
            $scope.meta = false;
        });
    }

    $http({
        method: 'GET',
        url: '/api/info'
    }).then(function successCallback(response) {
        $scope.info = response.data
        $scope.searchGenes($scope.info.delim);
    }, function errorCallback(response) {
        console.error(response)
    });        

    $scope.calcNumSamples = function() {
        $scope.numSamples = 0;
        var metaNames = Object.keys($scope.meta)
        $scope.metaTypes = metaNames;
        var smallestKey
        var smallestKeyValues = null;
        var smallestKeyNum = Number.POSITIVE_INFINITY
        for (var i=0; i < metaNames.length; i++) {
            var metaValues = Object.keys($scope.meta[metaNames[i]]);
            if (metaValues.length < smallestKeyNum) {
                smallestKeyNum = metaValues.length
                smallestKeyValues = metaValues
                smallestKey = metaNames[i]
            }
        }
        for (var i=0; i < smallestKeyValues.length; i++) {
            $scope.numSamples += $scope.meta[smallestKey][smallestKeyValues[i]]
        }
        if ($scope.totalNumSamples === null) {
            $scope.totalNumSamples = $scope.numSamples;
        }
        console.log($scope.numSamples)
    }

    $scope.buildChartData = function(changed) {
        if ($scope.metaType.current != null) {
            $scope.labels = [];
            $scope.values = [];
            angular.forEach($scope.meta[$scope.metaType.current],function(value,key) {
                $scope.labels.push(key);
                $scope.values.push(value);
            },null);
            if (changed === true) {
                $scope.metaType.values = Object.keys($scope.allMeta[$scope.metaType.current]);
                if ($scope.selectedValues[$scope.metaType.current] != undefined) {
                    $scope.metaType.selected = $scope.selectedValues[$scope.metaType.current];
                } else {
                    $scope.metaType.selected = [];
                }
                
            }
        }        
    }

    $scope.clickChart = function(x) {
        if (x.length > 0) {
            var clickedVal = $scope.labels[x[0]._index]
            var index = $scope.metaType.selected.indexOf(clickedVal);
            console.log(clickedVal,index)
            if (index == -1) {
                $scope.metaType.selected.push(clickedVal);
            } else {
                $scope.metaType.selected.splice(index,1);
            }
            $scope.$apply();
        }
    }

    $scope.commitFilters = function() {
        $scope.selectedValues[$scope.metaType.current] = $scope.metaType.selected
    }

    $scope.selectFilter = function(val) {
        if (val != $scope.metaType.current) {
            $scope.metaType.current = val;
            $scope.buildChartData(true);
        }
    }

    $scope.deleteFilter = function(val) {
        delete $scope.selectedValues[val];
        if ($scope.metaType.current == val) {
            $scope.metaType.selected = [];
        }
    }

    $scope.noFilters = function() {
        return Object.keys($scope.selectedValues).length === 0;
    }

    $scope.numberOfSamples = function(item) {
        var value = $scope.meta[$scope.metaType.current][item]
        if (value !== undefined) {
            return " : " + value;
        } else {
            return "";
        }
    }

    $scope.logScope = function() {
        console.log($scope);
    }

    $scope.searchGenes = function(query) {
        if (query.length == 0) {
            query = $scope.info.delim;
        }
        $http({
            method: 'GET',
            url: '/api/genes?query='+query
        }).then(function successCallback(response) {
            console.log(response)
            $scope.genes.results = response.data.genes;
        }, function errorCallback(response) {
            console.error(response);
        });
    }

    $scope.submitQuery = function() {
        var data = angular.merge({},$scope.selectedValues,{"genes":$scope.genes.selected});
        console.log(data);
        $scope.queryValue = JSON.stringify(data);
    }

    $scope.nextRoute = function() {
        $location.path($scope.targetRoute);
    }

    $rootScope.$on('$routeChangeSuccess', function(scope, current, pre) {
        console.log($location.path())

        switch($location.path()) {
            case "/":
                $scope.btnText = "Next <i class='fa fa-arrow-right'></i>";
                $scope.targetRoute = "/samples";
                break;
            case "/samples":
                $scope.btnText = "Next <i class='fa fa-arrow-right'></i>";
                $scope.targetRoute = "/genes";
                break;
            case "/genes":
                $scope.btnText = "Submit";
                $scope.targetRoute = "/download";
                break;
            default:
                $scope.btnText = "<i class='fa fa-home'></i>";
                $scope.targetRoute = "/";
                break;
        }

    });

}]);