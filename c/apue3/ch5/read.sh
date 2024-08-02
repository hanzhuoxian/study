
策略中心
grep  'ExperStrategyCenter.*remote_ip' /home/work/cashdesk/log/ral/ral-worker.log

apollo
grep 'apollo_config.*remote_ip' /home/work/cashdesk/log/ral/ral-worker.log

crossborder

access-gateway.pay
grep 'access-gateway.*remote_ip' /home/work/cashdesk/log/ral/ral-worker.log
pdo -b wallet-cashdesk.pay -r 321 -y -c "grep -a 'access-gateway'  log/ral/ral-worker.log.*"

# facepayserver
facepay
grep facepay /home/work/cashdesk/log/ral/ral-worker.log.*
grep 'facepay.*remote_ip' /home/work/cashdesk/log/ral/ral-worker.log
pdo -b wallet-cashdesk.pay -r 321 -y -c "grep -a 'facepay'  log/ral/ral-worker.log.*"

# faceimageserver
face_image_server
grep 'face_image_server.*remote_ip' /home/work/cashdesk/log/ral/ral-worker.log
grep face_image_server /home/work/cashdesk/log/ral/ral-worker.log.*
pdo -b wallet-cashdesk.pay -r 321 -y -c "grep -a 'face_image_server'  log/ral/ral-worker.log.*"

# facilepayserver
facilepayserver
grep 'facilepaycenter.*o2oserver' /home/work/cashdesk/log/ral/ral-worker.log.*
pdo -b wallet-cashdesk.pay -r 321 -y -c "grep -a 'facilepaycenter'  log/ral/ral-worker.log.*"

# batchrecv-server
batch_recv_server
grep batch_recv_server /home/work/cashdesk/log/ral/ral-worker.log.*
pdo -b wallet-cashdesk.pay -r 321 -y -c "grep -a 'batch_recv_server'  log/ral/ral-worker.log.*"

 grep '/usercenter/format/auth.*checkCanBeAuthed' /home/work/cashdesk/log/cashdesk/cashdesk.log.2021111121

pdo -b wallet-cashdesk.pay -r 321 -y -c "grep -a '/usercenter/format/auth.*checkCanBeAuthed'  log/cashdesk/cashdesk.log.2021111121"

C级
【功能】
1. 账户收银台数据同步
2. 活体校验人像分数值
【模块】
wallet-cashdesk/cashdesk 0.0.48.5
pay-common/odp-phplib 1.0.77
【参与人】
RD 韩剑 QA 徐梦、牛毅
【操作单】
http://noah.duxiaoman-int.com/bid#/task/detail?jobId=127932&productName=pay&systemName=cashdesk&appName=wallet-cashdesk
@OP-金云 @OP-闫彩 辛苦过下单

http://anypay.duxiaoman-int.com/success.html?bank_no=&bfb_order_create_time=20211111200715&bfb_order_no=2021111110001938211031749034215&buyer_sp_username=&contract_no=202111111000774365&currency=1&extra=&fee_amount=0&input_charset=1&order_no=1637069227891&pay_result=1&pay_time=20211111200730&pay_type=1&sign_method=1&sp_no=1000193821&total_amount=10&transport_amount=0&unit_amount=10&unit_count=1&version=2&sign=9e98580660cadcbe9fa9b8e68cdfbc3c


{"5271":{"agreementNoPreg":"","agreeNoReplacement":""}, "5268":{"agreementNoPreg":"","agreeNoReplacement":""}}