import coreModule from 'app/core/core_module';

const template = `
<div class="modal-body">
  <div class="modal-header">
    <h2 class="modal-header-title">
      <i class="fa fa-exclamation"></i>
      <span class="p-l-1">Ungespeicherte Änderungen</span>
    </h2>

    <a class="modal-header-close" ng-click="ctrl.dismiss();">
      <i class="fa fa-remove"></i>
    </a>
  </div>

  <div class="modal-content text-center">

    <div class="confirm-modal-text">
      Wollen Sie Ihre Änderungen speichern?
    </div>

    <div class="confirm-modal-buttons">
      <button type="button" class="btn btn-success" ng-click="ctrl.save()">Speichern</button>
      <button type="button" class="btn btn-danger" ng-click="ctrl.discard()">Verwerfen</button>
      <button type="button" class="btn btn-inverse" ng-click="ctrl.dismiss()">Abbrechen</button>
    </div>
  </div>
</div>
`;

export class UnsavedChangesModalCtrl {
  clone: any;
  dismiss: () => void;

  /** @ngInject */
  constructor(private unsavedChangesSrv) {}

  discard() {
    this.dismiss();
    this.unsavedChangesSrv.tracker.discardChanges();
  }

  save() {
    this.dismiss();
    this.unsavedChangesSrv.tracker.saveChanges();
  }
}

export function unsavedChangesModalDirective() {
  return {
    restrict: 'E',
    template: template,
    controller: UnsavedChangesModalCtrl,
    bindToController: true,
    controllerAs: 'ctrl',
    scope: { dismiss: '&' },
  };
}

coreModule.directive('unsavedChangesModal', unsavedChangesModalDirective);
