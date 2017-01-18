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

app.controller("MainController",['$scope','$http','$timeout',function($scope,$http,$timeout) {
    $scope.query = "";
    $scope.meta = null;
    $scope.metaTypes = null;
    $scope.genes = null;
    $scope.selectedValues = {};
    $scope.totalNumSamples = null;
    $scope.totalNumGenes = null;
    $scope.numSamples = 0;
    $scope.numCalculating = 0;

    $scope.metaType = {
        "current": null,
        "values": [],
        "selected":[]
    };

    $scope.labels = ["one","two","three"];
    $scope.values = [500,350,212];

    $timeout(function() {
        $scope.values = [700,150,420];
    },3000)


    $scope.$watch("selectedValues", function(newValue,oldValue) {
        angular.forEach(newValue,function(value,key) {
            if (value.length == 0) {
                delete this[key]
            }
        },newValue)
        $scope.query = JSON.stringify(newValue);
        $scope.fetchMeta();
    }, true);

    $scope.fetchMeta = function() {
        console.log("fetching meta",$scope.query)
        $http({
            method: 'GET',
            url: '/meta?query='+$scope.query
        }).then(function successCallback(response) {
            console.log(response.data);
            $scope.meta = response.data;
            $scope.calcNumSamples();

            $scope.buildChartData(false);
        }, function errorCallback(response) {
            console.error(response)
            $scope.meta = false;
        });
    }

    $http({
        method: 'GET',
        url: '/info'
    }).then(function successCallback(response) {
        console.log(response.data);
        $scope.totalNumGenes = response.data.genes
    }, function errorCallback(response) {
        console.error(response)
    });        

    $scope.fetchMetaValues = function(name) {
        $http({
            method: 'GET',
            url: '/meta?q='+name
        }).then(function successCallback(response) {
            console.log(response);
        }, function errorCallback(response) {
            console.error(response);
            alert("Error fetching values");
        })
    }

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
                $scope.metaType.values = angular.copy($scope.labels);
                $scope.metaType.selected = [];
            }
        }        
    }



    $scope.clickChart = function(x) {
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

    $scope.logScope = function() {
        console.log($scope);
    }

}]);