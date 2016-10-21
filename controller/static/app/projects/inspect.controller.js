(function(){
    'use strict';

    angular
        .module('shipyard.projects')
        .controller('InspectController', InspectController);

    InspectController.$inject = ['resolvedResults', '$scope' , '$rootScope', 'ProjectService', '$interval', 'RegistryService', '$stateParams'];
    function InspectController(resolvedResults, $scope, $rootScope, ProjectService, $interval,RegistryService, $stateParams) {

        var vm = this;

        $rootScope.$on('$stateChangeStart',
            function(){
                ProjectService.cancel();
                vm.refresh = false;
                if(angular.isDefined(timer)) {
                    $interval.cancel(timer);
                    timer=undefined;
                }
            });

        vm.refresh = false;
        var timer = undefined;

        $scope.$on('ngRepeatFinished', function() {
            $('.ui.sortable.celled.table').tablesort();
        });

        vm.showProjectHistory = showProjectHistory;
        vm.showTestHistory = showTestHistory;
        vm.showTestResults = showTestResults;
        vm.closeModal = closeModal;
        vm.testResults = testResults;
        vm.startRefresh = startRefresh;
        vm.cancelRefresh = cancelRefresh;

        vm.selectedId = null;
        vm.selectedTestName = null;
        vm.results = resolvedResults;
        vm.proj = ProjectService.getProjectByID(vm.results.projectId);

        angular.forEach(vm.results.testResults, function (result, key) {
            testResults(vm.results.projectId,result.testId,result.buildId).then(function (response) {
                vm.results[key].istestResult = response;
            })
        });

        function showProjectHistory() {
            $('#inspect-project-history-' + vm.results.projectId)
                .sidebar('toggle')
            ;
        }

        function showTestHistory() {
            $('#inspect-test-history-' + vm.results.projectId)
                .sidebar('toggle')
            ;
        }
        function testResults(projectId, testId, buildId) {
            return ProjectService.buildResults(projectId, testId, buildId)
                .then(function(data) {
                    return true;
                }, function(data) {
                    return false;
                });
        }

        function startRefresh(projectId) {
            vm.refresh = true;
            ProjectService.results(projectId)
                .then(function(data) {
                    vm.results = data;
                }, function(data) {
                    vm.error = data;
                });
            timer = $interval(function(){
                ProjectService.results(projectId)
                    .then(function(data) {
                        vm.results = data;
                    }, function(data) {
                        vm.error = data;
                    });
            },10000);
        }

        function cancelRefresh() {
            vm.refresh = false;
            if(angular.isDefined(timer)) {
                $interval.cancel(timer);
                timer=undefined;
            }
        }

        function showTestResults(id, name) {
            vm.selectedId = id;
            vm.selectedTestName = name;
            $('#inspect-test-history-' + vm.results.projectId)
                .sidebar('toggle');

            $(".ui.fullscreen.modal.transition.view.results").modal('show');
        }

        function closeModal() {
            $(".ui.fullscreen.modal.transition.view.results").modal('hide');
        }

    }
})();
