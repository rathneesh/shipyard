(function(){
    'use strict';

    angular
        .module('shipyard.projects')
        .controller('BuildResultsController', BuildResultsController);

    BuildResultsController.$inject = ['buildResultsTable', '$scope', 'ProjectService', '$stateParams'];
    function BuildResultsController(buildResultsTable, $scope, ProjectService, $stateParams) {
        var vm = this;

        vm.finalResults = buildResultsTable.data.FinalResults;
        vm.buildResults = buildResultsTable.data.BuildResults;
        vm.artifact = buildResultsTable.data.Artifact;
        vm.vulnerabilities = [];
        vm.noOfVulnerabilitiesForFeatures = {};
        vm.noOfVulnerabilitiesForLayers = {};

        for (var layer in vm.buildResults) {
            var showLayer = true;
            vm.noOfVulnerabilitiesForLayers[layer] = 0;
            for (var feature in vm.buildResults[layer]){
                var showFeature = true;
                for (var i=0; i < vm.buildResults[layer][feature].length; i++){
                    vm.vulnerabilities.push({"layer":layer, "feature":feature, "vulnerability":vm.buildResults[layer][feature][i], "showLayer":showLayer, "showFeature":showFeature});
                    showFeature = false;
                    showLayer = false;
                }
                vm.noOfVulnerabilitiesForFeatures[feature] = vm.buildResults[layer][feature].length;
                vm.noOfVulnerabilitiesForLayers[layer] += vm.buildResults[layer][feature].length;
            }

        }

    }


})();
