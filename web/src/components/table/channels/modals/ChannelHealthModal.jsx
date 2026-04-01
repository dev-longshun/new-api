import React, { useEffect, useState } from 'react';
import { Modal, Table, Tag, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, timestamp2string } from '../../../../helpers';

const renderResponseTimeTag = (ms, t) => {
  if (!ms && ms !== 0) return <Tag color='grey'>{t('未测试')}</Tag>;
  if (ms <= 1000) return <Tag color='green'>{ms}ms</Tag>;
  if (ms <= 3000) return <Tag color='lime'>{ms}ms</Tag>;
  if (ms <= 5000) return <Tag color='yellow'>{ms}ms</Tag>;
  return <Tag color='red'>{ms}ms</Tag>;
};

const ChannelHealthModal = ({ visible, channelId, channelName, onClose }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);

  const fetchData = async () => {
    if (!channelId) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/channel/${channelId}/health?limit=50`);
      if (res.data.success) {
        setData(res.data.data || []);
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (visible && channelId) {
      fetchData();
    }
  }, [visible, channelId]);
  const columns = [
    {
      title: t('时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (val) => timestamp2string(val),
    },
    {
      title: t('响应时间'),
      dataIndex: 'response_time',
      key: 'response_time',
      render: (val) => renderResponseTimeTag(val, t),
    },
    {
      title: t('状态'),
      dataIndex: 'success',
      key: 'success',
      render: (val) =>
        val ? (
          <Tag color='green'>{t('成功')}</Tag>
        ) : (
          <Tag color='red'>{t('失败')}</Tag>
        ),
    },
    {
      title: t('错误信息'),
      dataIndex: 'error_message',
      key: 'error_message',
      ellipsis: true,
      render: (val) => val || '-',
    },
  ];

  const successCount = data.filter((d) => d.success).length;
  const successRate = data.length > 0 ? ((successCount / data.length) * 100).toFixed(1) : 0;

  return (
    <Modal
      title={`${t('健康历史')} - ${channelName || channelId}`}
      visible={visible}
      onCancel={onClose}
      footer={null}
      width={700}
    >
      <div style={{ marginBottom: 12 }}>
        <Tag color={successRate >= 90 ? 'green' : successRate >= 70 ? 'yellow' : 'red'}>
          {t('成功率')}: {successRate}% ({successCount}/{data.length})
        </Tag>
      </div>
      <Spin spinning={loading}>
        <Table
          columns={columns}
          dataSource={data}
          pagination={{ pageSize: 10 }}
          size='small'
          rowKey='id'
          empty={t('暂无数据')}
        />
      </Spin>
    </Modal>
  );
};

export default ChannelHealthModal;
