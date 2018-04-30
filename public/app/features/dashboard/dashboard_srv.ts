import coreModule from 'app/core/core_module';
import { DashboardModel } from './dashboard_model';
import locationUtil from 'app/core/utils/location_util';

export class DashboardSrv {
  dash: any;

  /** @ngInject */
  constructor(private backendSrv, private $rootScope, private $location) {}

  create(dashboard, meta) {
    return new DashboardModel(dashboard, meta);
  }

  setCurrent(dashboard) {
    this.dash = dashboard;
  }

  getCurrent() {
    return this.dash;
  }

  handleSaveDashboardError(clone, options, err) {
    options = options || {};
    options.overwrite = true;

    if (err.data && err.data.status === 'version-mismatch') {
      err.isHandled = true;

      this.$rootScope.appEvent('confirm-modal', {
        title: 'Konflikt',
        text: 'Jemand anderes hat dieses Dashboard aktualisiert.',
        text2: 'Möchten Sie dieses Dashboard trotzdem speichern?',
        yesText: 'Speichern & Überschreiben',
        icon: 'fa-warning',
        onConfirm: () => {
          this.save(clone, options);
        },
      });
    }

    if (err.data && err.data.status === 'name-exists') {
      err.isHandled = true;

      this.$rootScope.appEvent('confirm-modal', {
        title: 'Konflikt',
        text: 'Ein Dashboard mit demselben Namen im ausgewählten Ordner existiert bereits.',
        text2: 'Wollen Sie das Dashboard trotzdem speichern?',
        yesText: 'Speichern & Überschreiben',
        icon: 'fa-warning',
        onConfirm: () => {
          this.save(clone, options);
        },
      });
    }

    if (err.data && err.data.status === 'plugin-dashboard') {
      err.isHandled = true;

      this.$rootScope.appEvent('confirm-modal', {
        title: 'Plugin Dashboard',
        text: err.data.message,
        text2: 'Die Änderungen gehen verloren, wenn das Plugin aktualisiert wird. Benutzen Sie Speichern unter, zum erstellen einer eigenen Vers.',
        yesText: 'Überschreiben',
        icon: 'fa-warning',
        altActionText: 'Speichern als',
        onAltAction: () => {
          this.showSaveAsModal();
        },
        onConfirm: () => {
          this.save(clone, { overwrite: true });
        },
      });
    }
  }

  postSave(clone, data) {
    this.dash.version = data.version;

    const newUrl = locationUtil.stripBaseFromUrl(data.url);
    const currentPath = this.$location.path();

    if (newUrl !== currentPath) {
      this.$location.url(newUrl).replace();
    }

    this.$rootScope.appEvent('dashboard-saved', this.dash);
    this.$rootScope.appEvent('alert-success', ['Dashboard saved']);

    return this.dash;
  }

  save(clone, options) {
    options = options || {};
    options.folderId = options.folderId >= 0 ? options.folderId : this.dash.meta.folderId || clone.folderId;

    return this.backendSrv
      .saveDashboard(clone, options)
      .then(this.postSave.bind(this, clone))
      .catch(this.handleSaveDashboardError.bind(this, clone, options));
  }

  saveDashboard(options, clone) {
    if (clone) {
      this.setCurrent(this.create(clone, this.dash.meta));
    }

    if (!this.dash.meta.canSave && options.makeEditable !== true) {
      return Promise.resolve();
    }

    if (this.dash.title === 'New dashboard') {
      return this.showSaveAsModal();
    }

    if (this.dash.version > 0) {
      return this.showSaveModal();
    }

    return this.save(this.dash.getSaveModelClone(), options);
  }

  showSaveAsModal() {
    this.$rootScope.appEvent('show-modal', {
      templateHtml: '<save-dashboard-as-modal dismiss="dismiss()"></save-dashboard-as-modal>',
      modalClass: 'modal--narrow',
    });
  }

  showSaveModal() {
    this.$rootScope.appEvent('show-modal', {
      templateHtml: '<save-dashboard-modal dismiss="dismiss()"></save-dashboard-modal>',
      modalClass: 'modal--narrow',
    });
  }

  starDashboard(dashboardId, isStarred) {
    let promise;

    if (isStarred) {
      promise = this.backendSrv.delete('/api/user/stars/dashboard/' + dashboardId).then(() => {
        return false;
      });
    } else {
      promise = this.backendSrv.post('/api/user/stars/dashboard/' + dashboardId).then(() => {
        return true;
      });
    }

    return promise.then(res => {
      if (this.dash && this.dash.id === dashboardId) {
        this.dash.meta.isStarred = res;
      }
      return res;
    });
  }
}

coreModule.service('dashboardSrv', DashboardSrv);
