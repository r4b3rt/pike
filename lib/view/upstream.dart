///
/// upstream配置页
///
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/bloc.dart';
import '../config/application.dart';
import '../helper/util.dart';
import '../model/config.dart';
import '../widget/button.dart';
import '../widget/error_message.dart';

@immutable
class UpstreamPage extends StatefulWidget {
  const UpstreamPage({
    Key key,
  }) : super(key: key);
  @override
  _UpstreamPageState createState() => _UpstreamPageState();
}

final policyList = <String>['roundRobin', 'random', 'first', 'leastconn'];

class _ServerEditor {
  final TextEditingController addrController = TextEditingController();
  bool backup = false;
}

class _UpstreamPageState extends State<UpstreamPage> {
  final GlobalKey _formKey = GlobalKey<FormState>();

  final TextEditingController _nameController = TextEditingController();
  final TextEditingController _healthCheckController = TextEditingController();
  final TextEditingController _acceptEncodingController =
      TextEditingController();
  final TextEditingController _remarkController = TextEditingController();
  final _serverEditors = <_ServerEditor>[];

  String _policy = policyList.first;
  bool _enableH2C = false;

  String _mode = '';
  final _editMode = 'eidt';
  final _updateMode = 'update';

  ConfigBloc _configBloc;

  @override
  void initState() {
    super.initState();
    _configBloc = BlocProvider.of<ConfigBloc>(context);
  }

  bool get _isEditting => _mode.isNotEmpty;

  bool get _isUpdateding => _mode == _updateMode;

  // _reset 重置表单所有元素
  void _reset() {
    _nameController.clear();
    _healthCheckController.clear();
    _acceptEncodingController.clear();
    _remarkController.clear();
    _policy = policyList.first;
    _enableH2C = false;
    _serverEditors.clear();
    _serverEditors.add(_ServerEditor());
  }

  // _fillTextEditor 填充编辑数据
  void _fillTextEditor(UpstreamConfig element) {
    _nameController.value = TextEditingValue(text: element.name ?? '');
    _healthCheckController.value =
        TextEditingValue(text: element.healthCheck ?? '');
    _acceptEncodingController.value =
        TextEditingValue(text: element.acceptEncoding ?? '');
    _remarkController.value = TextEditingValue(text: element.remark ?? '');
    _policy = element.policy ?? policyList.first;
    _enableH2C = element.enableH2C ?? false;
    _serverEditors.clear();
    element.servers?.forEach((element) {
      final s = _ServerEditor();
      s.addrController.value = TextEditingValue(text: element.addr ?? '');
      s.backup = element.backup ?? false;
      _serverEditors.add(s);
    });
  }

  // _createRowItem 生成表单元素
  Widget _createRowItem(String text) => Padding(
        padding: EdgeInsets.only(
          top: Application.defaultPadding,
          bottom: Application.defaultPadding,
        ),
        child: Text(
          text ?? '--',
          textAlign: TextAlign.center,
        ),
      );

  void _deleteUpstream(ConfigCurrentState state, String name) {
    // 校验该upstream是否被其它配置使用
    if (!state.config.validateForDelete('upstream', name)) {
      showErrorMessage('$name is used, it cant not be deleted');
      return;
    }
    final upstreamList = <UpstreamConfig>[];
    state.config.upstreams?.forEach((element) {
      if (element.name != name) {
        upstreamList.add(element);
      }
    });
    // 更新配置
    _configBloc.add(ConfigUpdate(
      config: state.config.copyWith(
        upstreams: upstreamList,
      ),
    ));
  }

  // _addUpstream 添加upstream配置，如果有当前相同配置，则替换
  void _addUpstream(ConfigCurrentState state) {
    final name = _nameController.text?.trim();
    final healthCheck = _healthCheckController.text?.trim();
    final acceptEncoding = _acceptEncodingController.text?.trim();
    final remark = _remarkController.text?.trim();

    final servers = <UpstreamServerConfig>[];
    _serverEditors.forEach((element) {
      final addr = element.addrController.text?.trim();
      if (addr != null && addr.isNotEmpty) {
        servers.add(UpstreamServerConfig(
          addr: addr,
          backup: element.backup,
        ));
      }
    });

    final upstreamConfig = UpstreamConfig(
      name: name,
      healthCheck: healthCheck,
      policy: _policy,
      enableH2C: _enableH2C,
      acceptEncoding: acceptEncoding,
      servers: servers,
      remark: remark,
    );
    final upstreamList = <UpstreamConfig>[];
    state.config.upstreams?.forEach((element) {
      if (element.name != name) {
        upstreamList.add(element);
      }
    });
    upstreamList.add(upstreamConfig);
    _configBloc.add(ConfigUpdate(
      delay: '2s',
      config: state.config.copyWith(
        upstreams: upstreamList,
      ),
    ));
    // 重置当前模式
    setState(() {
      _mode = '';
    });
  }

  // _renderServerList 渲染服务器列表
  Widget _renderServerList(List<UpstreamServerConfig> servers) {
    final items = servers?.map((element) {
      var addr = element.addr;
      if (element.backup != null && element.backup) {
        addr += ' (backup)';
      }
      var icon = Icon(
        Icons.check,
        color: Colors.green,
        size: Application.defaultFontSize,
      );
      if (element.healthy == null || !element.healthy) {
        icon = Icon(
          Icons.close,
          color: Colors.red,
          size: Application.defaultFontSize,
        );
      }
      return Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(addr),
          Container(
            width: 5,
          ),
          icon,
        ],
      );
    })?.toList();
    return Padding(
      padding: EdgeInsets.only(
        top: Application.defaultPadding,
        bottom: Application.defaultPadding,
      ),
      child: Column(
        children: items,
      ),
    );
  }

  // _renderServerEditor 渲染服务器编辑列表
  Widget _renderServerEditor() {
    // 生成服务列表
    List<Widget> servers = _serverEditors.map((element) {
      // 地址
      final addr = Container(
        child: TextFormField(
          controller: element.addrController,
          decoration: InputDecoration(
            labelText: 'Addr',
            hintText:
                'Please input the server addr, eg: http://127.0.0.1:3015 ',
          ),
          validator: (v) {
            if (v == null || v.isEmpty) {
              return null;
            }
            if (RegExp(r'^http(s?)://').hasMatch(v)) {
              return null;
            }
            return 'Server addr should be http://xxx or https://xxx';
          },
        ),
      );
      // 是否备份节点
      final backup = Row(
        children: [
          Text('Backup'),
          Container(
            width: Application.defaultPadding,
          ),
          Switch(
            value: element.backup,
            onChanged: (bool value) {
              setState(() {
                element.backup = value;
              });
            },
          ),
        ],
      );

      return Container(
        decoration: BoxDecoration(
          border: Border.all(
            color: Application.defaultBorderColor,
          ),
        ),
        margin: EdgeInsets.only(
          top: Application.defaultPadding,
        ),
        padding: EdgeInsets.all(Application.defaultPadding),
        child: Column(
          children: [
            addr,
            backup,
          ],
        ),
      );
    }).toList();
    // 添加标题
    servers.insert(
        0,
        Container(
          margin: EdgeInsets.only(
            top: Application.defaultPadding,
          ),
          width: double.infinity,
          child: Row(
            children: [
              Text('Servers'),
              Text(
                '(upstream server list)',
                style: TextStyle(
                  color: Application.fontColorOfSecondaryColor,
                  fontSize: Application.smallFontSize,
                ),
              )
            ],
          ),
        ));
    // 添加服务器按钮
    servers.add(Container(
      child: XFullButton(
        text: Text('Add More Server'),
        padding: EdgeInsets.all(1.5 * Application.defaultPadding),
        margin: EdgeInsets.only(
          top: Application.defaultPadding,
        ),
        onPressed: () {
          setState(() {
            _serverEditors.add(_ServerEditor());
          });
        },
      ),
    ));
    return Column(
      children: servers,
    );
  }

  // _renderEditor 渲染编辑表单
  Widget _renderEditor() {
    if (!_isEditting) {
      return Container();
    }
    final formItems = <Widget>[];

    // 名称
    formItems.add(TextFormField(
      autofocus: true,
      readOnly: _isUpdateding,
      controller: _nameController,
      decoration: InputDecoration(
        labelText: 'Name',
        hintText: 'Please input the name of upstream',
      ),
      validator: (v) => v.trim().isNotEmpty ? null : 'Name can not be null',
    ));

    // health check
    formItems.add(TextFormField(
      controller: _healthCheckController,
      decoration: InputDecoration(
        labelText: 'Health Check',
        hintText: 'Please input the health check url, eg: /ping',
      ),
      validator: (v) {
        // 允许不配置
        if (v.trim().isEmpty) {
          return null;
        }
        if (!v.startsWith('/')) {
          return 'Health check should be url path';
        }
        return null;
      },
    ));

    // policy选择器
    formItems.add(Row(
      children: [
        Text('Policy'),
        Container(
          width: Application.defaultPadding,
        ),
        DropdownButton(
            value: _policy ?? policyList.first,
            items: policyList
                .map<DropdownMenuItem<String>>(
                    (String value) => DropdownMenuItem<String>(
                          value: value,
                          child: Text(value),
                        ))
                .toList(),
            onChanged: (String newValue) {
              setState(() {
                _policy = newValue;
              });
            }),
      ],
    ));

    // 是否启用 h2c
    formItems.add(Row(
      children: [
        Text('Enable H2C'),
        Container(
          width: Application.defaultPadding,
        ),
        Switch(
          value: _enableH2C,
          onChanged: (bool value) {
            setState(() {
              _enableH2C = value;
            });
          },
        ),
      ],
    ));

    // accept encoding
    formItems.add(TextFormField(
      controller: _acceptEncodingController,
      decoration: InputDecoration(
        labelText: 'Accept Encoding',
        hintText:
            'Please input the accept encoding of proxy, eg: gzip, br [optional]',
      ),
    ));

    formItems.add(_renderServerEditor());

    // remark
    formItems.add(TextFormField(
      controller: _remarkController,
      minLines: 3,
      maxLines: 3,
      decoration: InputDecoration(
        labelText: 'Remark',
        hintText: 'Please input the remark for upstream',
      ),
    ));

    return Container(
      margin: EdgeInsets.only(
        top: 3 * Application.defaultPadding,
      ),
      child: Form(
        key: _formKey, //设置globalKey，用于后面获取FormState
        child: Column(
          children: formItems,
        ),
      ),
    );
  }

  // _renderUpstreamList 渲染upstream列表
  Widget _renderUpstreamList(ConfigCurrentState state) {
    // 表头
    final rows = <TableRow>[
      TableRow(
        children: [
          _createRowItem('Name'),
          _createRowItem('Health Check'),
          _createRowItem('Policy'),
          _createRowItem('Enable H2C'),
          _createRowItem('Accept Encoding'),
          _createRowItem('Servers'),
          _createRowItem('Remark'),
          _createRowItem('Operations'),
        ],
      ),
    ];
    state.config.upstreams?.forEach((element) {
      var enableH2C = 'off';
      if (element.enableH2C != null && element.enableH2C) {
        enableH2C = 'on';
      }
      rows.add(TableRow(
        children: [
          _createRowItem(element.name),
          _createRowItem(element.healthCheck ?? ''),
          _createRowItem(element.policy ?? ''),
          _createRowItem(enableH2C),
          _createRowItem(element.acceptEncoding ?? ''),
          _renderServerList(element.servers),
          _createRowItem(element.remark),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              TextButton(
                onPressed: () {
                  // 重置当前数据，并将需要更新的配置填充
                  _reset();
                  _fillTextEditor(element);

                  setState(() {
                    _mode = _updateMode;
                  });
                },
                child: Text('Update'),
              ),
              TextButton(
                onPressed: () {
                  _deleteUpstream(state, element.name);
                },
                child: Text('Delete'),
              ),
            ],
          ),
        ],
      ));
    });
    return Table(
      columnWidths: {
        // 指定表格列宽
        1: FixedColumnWidth(150),
        2: FixedColumnWidth(120),
        3: FixedColumnWidth(100),
        4: FixedColumnWidth(140),
        7: FixedColumnWidth(150),
      },
      border: TableBorder.all(
        color: Application.primaryColor.withAlpha(60),
      ),
      children: rows,
    );
  }

  @override
  Widget build(BuildContext context) =>
      BlocBuilder<ConfigBloc, ConfigState>(builder: (context, state) {
        if (state is ConfigErrorState) {
          return XErrorMessage(
            message: state.message,
            title: 'Get upstream config fail',
          );
        }
        final currentConfig = state as ConfigCurrentState;
        var btnText = _isEditting ? 'Save Upstream' : 'Add Upstream';
        if (currentConfig.isProcessing) {
          btnText = 'Processing...';
        }
        return SingleChildScrollView(
          child: Container(
            margin: EdgeInsets.all(3 * Application.defaultPadding),
            child: Column(
              children: [
                _renderUpstreamList(currentConfig),
                _renderEditor(),
                XFullButton(
                  margin: EdgeInsets.only(
                    top: 3 * Application.defaultPadding,
                  ),
                  onPressed: () {
                    if (currentConfig.isProcessing) {
                      return;
                    }
                    // 如果是编辑模式，则是添加或更新
                    if (_isEditting) {
                      if ((_formKey.currentState as FormState).validate()) {
                        _addUpstream(currentConfig);
                      }
                      return;
                    }
                    // 重置所有数据，设置为编辑模式
                    _reset();
                    setState(() {
                      _mode = _editMode;
                    });
                  },
                  text: Text(btnText),
                ),
              ],
            ),
          ),
        );
      });
}
